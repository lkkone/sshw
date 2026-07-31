package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSecretPrefersEnvironmentValue(t *testing.T) {
	t.Setenv("TEST_SECRET", " from-environment ")
	t.Setenv("TEST_SECRET_FILE", filepath.Join(t.TempDir(), "missing"))

	value, err := loadSecret("TEST_SECRET", "TEST_SECRET_FILE")
	if err != nil {
		t.Fatalf("loadSecret() error = %v", err)
	}
	if value != "from-environment" {
		t.Fatalf("loadSecret() = %q, want %q", value, "from-environment")
	}
}

func TestLoadSecretReadsFile(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretPath, []byte(" from-file \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_SECRET", "")
	t.Setenv("TEST_SECRET_FILE", secretPath)

	value, err := loadSecret("TEST_SECRET", "TEST_SECRET_FILE")
	if err != nil {
		t.Fatalf("loadSecret() error = %v", err)
	}
	if value != "from-file" {
		t.Fatalf("loadSecret() = %q, want %q", value, "from-file")
	}
}

func TestLoadSecretRejectsEmptyFile(t *testing.T) {
	secretPath := filepath.Join(t.TempDir(), "secret")
	if err := os.WriteFile(secretPath, []byte(" \n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_SECRET", "")
	t.Setenv("TEST_SECRET_FILE", secretPath)

	if _, err := loadSecret("TEST_SECRET", "TEST_SECRET_FILE"); err == nil {
		t.Fatal("loadSecret() error = nil, want an error")
	}
}
