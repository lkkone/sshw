package sshw

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"gopkg.in/yaml.v2"
)

func TestSyncRunnerDownloadsBacksUpAndCachesVersion(t *testing.T) {
	newConfig := []byte("- name: New host\n  alias: new\n  host: 10.0.0.2\n")
	sum := sha256.Sum256(newConfig)
	var requests atomic.Int32
	var conditionalRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.Header.Get("If-None-Match") == `"default-v2"` {
			conditionalRequests.Add(1)
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Header().Set("ETag", `"default-v2"`)
		w.Header().Set("X-SSHW-Version", "2")
		w.Header().Set("X-SSHW-SHA256", hex.EncodeToString(sum[:]))
		w.Write(newConfig)
	}))
	defer server.Close()

	home := t.TempDir()
	target := filepath.Join(home, ".sshw")
	oldConfig := []byte("- name: Old host\n  host: 10.0.0.1\n")
	if err := os.WriteFile(target, oldConfig, 0600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".sshw-sync.yaml")
	settings := SyncSettings{
		Version:         1,
		Server:          server.URL,
		Profile:         "default",
		Token:           "test-token",
		Target:          target,
		Backup:          true,
		BackupRetention: 5,
	}
	data, _ := yaml.Marshal(settings)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	runner := &SyncRunner{
		In:         strings.NewReader(""),
		Out:        &output,
		ErrOut:     &output,
		HTTPClient: server.Client(),
		HomeDir:    home,
		Now: func() time.Time {
			return time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
		},
	}
	if err := runner.Run([]string{"--config", configPath}); err != nil {
		t.Fatalf("first sync failed: %v\n%s", err, output.String())
	}
	got, _ := os.ReadFile(target)
	if !bytes.Equal(got, newConfig) {
		t.Fatalf("target = %q, want %q", got, newConfig)
	}
	backup, _ := os.ReadFile(target + ".backup-20260731-120000")
	if !bytes.Equal(backup, oldConfig) {
		t.Fatalf("backup = %q, want %q", backup, oldConfig)
	}

	output.Reset()
	if err := runner.Run([]string{"--config", configPath}); err != nil {
		t.Fatalf("second sync failed: %v", err)
	}
	if !strings.Contains(output.String(), "already up to date") {
		t.Fatalf("unexpected second sync output: %s", output.String())
	}
	if requests.Load() != 2 {
		t.Fatalf("request count = %d, want 2", requests.Load())
	}
	if conditionalRequests.Load() != 1 {
		t.Fatalf("conditional request count = %d, want 1", conditionalRequests.Load())
	}

	locallyModified := []byte("- name: Local edit\n  host: 10.0.0.99\n")
	if err := os.WriteFile(target, locallyModified, 0600); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	if err := runner.Run([]string{"--config", configPath}); err != nil {
		t.Fatalf("sync after local edit failed: %v\n%s", err, output.String())
	}
	got, _ = os.ReadFile(target)
	if !bytes.Equal(got, newConfig) {
		t.Fatalf("target after local edit = %q, want %q", got, newConfig)
	}
	if strings.Contains(output.String(), "already up to date") {
		t.Fatalf("local edit was incorrectly treated as current: %s", output.String())
	}
	if requests.Load() != 3 {
		t.Fatalf("request count after local edit = %d, want 3", requests.Load())
	}
	if conditionalRequests.Load() != 1 {
		t.Fatalf("local edit sent a stale ETag; conditional request count = %d, want 1", conditionalRequests.Load())
	}
}

func TestSyncRunnerDoesNotReplaceConfigWhenRemoteIsInvalid(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-SSHW-Version", "3")
		fmt.Fprintln(w, "- name: Broken")
	}))
	defer server.Close()

	home := t.TempDir()
	target := filepath.Join(home, ".sshw")
	original := []byte("- name: Safe\n  host: 10.0.0.1\n")
	if err := os.WriteFile(target, original, 0600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".sshw-sync.yaml")
	settings := SyncSettings{
		Version:         1,
		Server:          server.URL,
		Profile:         "default",
		Token:           "token",
		Target:          target,
		Backup:          true,
		BackupRetention: 5,
	}
	data, _ := yaml.Marshal(settings)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	runner := &SyncRunner{
		In:         strings.NewReader(""),
		Out:        io.Discard,
		ErrOut:     io.Discard,
		HTTPClient: server.Client(),
		HomeDir:    home,
		Now:        time.Now,
	}
	err := runner.Run([]string{"--config", configPath})
	if err == nil || !strings.Contains(err.Error(), "validation failed") {
		t.Fatalf("expected validation error, got %v", err)
	}
	got, _ := os.ReadFile(target)
	if !bytes.Equal(got, original) {
		t.Fatalf("local configuration changed: %q", got)
	}
}

func TestSyncStatusReportsLocallyModifiedConfiguration(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Fatalf("method = %s, want HEAD", r.Method)
		}
		w.Header().Set("X-SSHW-Version", "2")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	home := t.TempDir()
	target := filepath.Join(home, ".sshw")
	if err := os.WriteFile(target, []byte("- name: Local edit\n  host: 10.0.0.99\n"), 0600); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(home, ".sshw-sync.yaml")
	settings := SyncSettings{
		Version:         1,
		Server:          server.URL,
		Profile:         "default",
		Token:           "test-token",
		Target:          target,
		Backup:          true,
		BackupRetention: 5,
	}
	data, _ := yaml.Marshal(settings)
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := writeSyncState(syncStatePath(configPath), syncState{
		Server:    server.URL,
		Profile:   "default",
		ETag:      `"default-v2"`,
		Version:   2,
		SHA256:    strings.Repeat("0", sha256.Size*2),
		LocalPath: target,
	}); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	runner := &SyncRunner{
		In:         strings.NewReader(""),
		Out:        &output,
		ErrOut:     &output,
		HTTPClient: server.Client(),
		HomeDir:    home,
		Now:        time.Now,
	}
	if err := runner.Run([]string{"status", "--config", configPath}); err != nil {
		t.Fatalf("sync status failed: %v\n%s", err, output.String())
	}
	if !strings.Contains(output.String(), "local configuration changed; sync required") {
		t.Fatalf("unexpected status output: %s", output.String())
	}
}

func TestSyncInitClearsCachedStateWhenTargetChanges(t *testing.T) {
	home := t.TempDir()
	configPath := filepath.Join(home, ".sshw-sync.yaml")
	statePath := syncStatePath(configPath)
	if err := writeSyncState(statePath, syncState{
		Server:    "http://127.0.0.1:8080",
		Profile:   "default",
		ETag:      `"default-v1"`,
		Version:   1,
		LocalPath: filepath.Join(home, "old-target"),
	}); err != nil {
		t.Fatal(err)
	}

	var output bytes.Buffer
	runner := &SyncRunner{
		In:      strings.NewReader(""),
		Out:     &output,
		ErrOut:  &output,
		HomeDir: home,
		Now:     time.Now,
	}
	newTarget := filepath.Join(home, "new-target")
	err := runner.Run([]string{
		"init",
		"--config", configPath,
		"--target", newTarget,
		"--server", "http://127.0.0.1:8080",
		"--token", "test-token",
		"--allow-insecure",
	})
	if err != nil {
		t.Fatalf("sync init failed: %v\n%s", err, output.String())
	}
	if _, err := os.Stat(statePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cached state still exists: %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	var settings SyncSettings
	if err := yaml.UnmarshalStrict(data, &settings); err != nil {
		t.Fatal(err)
	}
	if settings.Target != newTarget {
		t.Fatalf("target = %q, want %q", settings.Target, newTarget)
	}
}
