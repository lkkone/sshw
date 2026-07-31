package sshw

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/atrox/homedir"
	"gopkg.in/yaml.v2"
)

const (
	syncConfigVersion  = 1
	defaultSyncProfile = "default"
)

var syncProfilePattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type SyncSettings struct {
	Version         int    `yaml:"version"`
	Server          string `yaml:"server"`
	Profile         string `yaml:"profile,omitempty"`
	Token           string `yaml:"token"`
	Target          string `yaml:"target,omitempty"`
	Backup          bool   `yaml:"backup"`
	BackupRetention int    `yaml:"backup-retention,omitempty"`
	AllowInsecure   bool   `yaml:"allow-insecure,omitempty"`
}

type syncState struct {
	Server    string    `json:"server"`
	Profile   string    `json:"profile"`
	ETag      string    `json:"etag"`
	Version   int       `json:"version"`
	SHA256    string    `json:"sha256"`
	SyncedAt  time.Time `json:"synced_at"`
	LocalPath string    `json:"local_path"`
}

type SyncRunner struct {
	In         io.Reader
	Out        io.Writer
	ErrOut     io.Writer
	HTTPClient *http.Client
	Now        func() time.Time
	HomeDir    string
}

func NewSyncRunner() *SyncRunner {
	return &SyncRunner{
		In:     os.Stdin,
		Out:    os.Stdout,
		ErrOut: os.Stderr,
		HTTPClient: &http.Client{
			Timeout: 20 * time.Second,
		},
		Now: time.Now,
	}
}

func RunSyncCommand(args []string) error {
	return NewSyncRunner().Run(args)
}

func (r *SyncRunner) Run(args []string) error {
	if r.In == nil {
		r.In = strings.NewReader("")
	}
	if r.Out == nil {
		r.Out = io.Discard
	}
	if r.ErrOut == nil {
		r.ErrOut = io.Discard
	}
	if r.HTTPClient == nil {
		r.HTTPClient = &http.Client{Timeout: 20 * time.Second}
	}
	if r.Now == nil {
		r.Now = time.Now
	}

	if len(args) > 0 {
		switch args[0] {
		case "init":
			return r.init(args[1:])
		case "status":
			return r.status(args[1:])
		case "help", "-h", "--help":
			r.printHelp()
			return nil
		}
	}
	return r.pull(args)
}

func (r *SyncRunner) printHelp() {
	fmt.Fprintln(r.Out, `Usage:
  sshw sync init [options]   create the sync settings
  sshw sync [options]        download and safely install the published config
  sshw sync status           show local and remote versions

Sync options:
  --config <path>            use another sync settings file
  --dry-run                  validate and preview without replacing the config
  --force                    download even when the version is unchanged`)
}

func (r *SyncRunner) init(args []string) error {
	flags := flag.NewFlagSet("sshw sync init", flag.ContinueOnError)
	flags.SetOutput(r.ErrOut)
	configPath := flags.String("config", "", "sync settings path")
	server := flags.String("server", "", "config server URL")
	token := flags.String("token", "", "device sync token")
	profile := flags.String("profile", defaultSyncProfile, "config profile")
	target := flags.String("target", "", "local sshw config path")
	allowInsecure := flags.Bool("allow-insecure", false, "allow plain HTTP")
	if err := flags.Parse(args); err != nil {
		return err
	}

	var err error
	if *configPath == "" {
		*configPath, err = r.defaultSyncConfigPath()
		if err != nil {
			return err
		}
	}
	if *target == "" {
		*target, err = r.defaultTargetPath()
		if err != nil {
			return err
		}
	}

	reader := bufio.NewReader(r.In)
	if strings.TrimSpace(*server) == "" {
		fmt.Fprint(r.Out, "Config server URL: ")
		*server, _ = reader.ReadString('\n')
	}
	if strings.TrimSpace(*token) == "" {
		fmt.Fprint(r.Out, "Device sync token: ")
		*token, _ = reader.ReadString('\n')
	}

	settings := SyncSettings{
		Version:         syncConfigVersion,
		Server:          strings.TrimRight(strings.TrimSpace(*server), "/"),
		Profile:         strings.TrimSpace(*profile),
		Token:           strings.TrimSpace(*token),
		Target:          strings.TrimSpace(*target),
		Backup:          true,
		BackupRetention: 5,
		AllowInsecure:   *allowInsecure,
	}
	if err := validateSyncSettings(&settings); err != nil {
		return err
	}
	data, err := yaml.Marshal(&settings)
	if err != nil {
		return err
	}
	if err := writePrivateFile(*configPath, data); err != nil {
		return fmt.Errorf("write sync settings: %w", err)
	}
	resolvedConfigPath, err := homedir.Expand(*configPath)
	if err != nil {
		return fmt.Errorf("resolve sync settings path: %w", err)
	}
	if err := os.Remove(syncStatePath(resolvedConfigPath)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("clear cached sync state: %w", err)
	}
	fmt.Fprintf(r.Out, "Sync settings saved to %s\n", *configPath)
	return nil
}

func (r *SyncRunner) pull(args []string) error {
	flags := flag.NewFlagSet("sshw sync", flag.ContinueOnError)
	flags.SetOutput(r.ErrOut)
	configPath := flags.String("config", "", "sync settings path")
	dryRun := flags.Bool("dry-run", false, "validate without installing")
	force := flags.Bool("force", false, "ignore the saved ETag")
	if err := flags.Parse(args); err != nil {
		return err
	}

	settings, resolvedConfigPath, err := r.loadSettings(*configPath)
	if err != nil {
		return err
	}
	target, err := homedir.Expand(settings.Target)
	if err != nil {
		return fmt.Errorf("expand target path: %w", err)
	}
	statePath := syncStatePath(resolvedConfigPath)
	state, _ := readSyncState(statePath)

	endpoint, err := syncEndpoint(settings)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+settings.Token)
	req.Header.Set("Accept", "application/yaml")
	req.Header.Set("User-Agent", "sshw-sync")
	if !*force &&
		state.ETag != "" &&
		state.Server == settings.Server &&
		state.Profile == settings.Profile &&
		state.LocalPath == target &&
		localConfigMatchesState(target, state) {
		req.Header.Set("If-None-Match", state.ETag)
	}

	fmt.Fprintf(r.Out, "Checking %s...\n", settings.Server)
	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to reach config server: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		fmt.Fprintf(r.Out, "Configuration is already up to date (version %d).\n", state.Version)
		return nil
	}
	if resp.StatusCode != http.StatusOK {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("config server returned %s: %s", resp.Status, strings.TrimSpace(string(message)))
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxConfigSize+1))
	if err != nil {
		return fmt.Errorf("download configuration: %w", err)
	}
	if len(data) > MaxConfigSize {
		return fmt.Errorf("remote configuration exceeds %d bytes", MaxConfigSize)
	}
	sum := sha256.Sum256(data)
	digest := hex.EncodeToString(sum[:])
	if expected := strings.TrimSpace(resp.Header.Get("X-SSHW-SHA256")); expected != "" && !strings.EqualFold(expected, digest) {
		return fmt.Errorf("remote configuration checksum mismatch")
	}
	stats, err := ValidateConfig(data)
	if err != nil {
		return fmt.Errorf("remote configuration validation failed: %w", err)
	}
	version, _ := strconv.Atoi(resp.Header.Get("X-SSHW-Version"))
	etag := resp.Header.Get("ETag")

	if *dryRun {
		fmt.Fprintf(r.Out, "Remote configuration is valid: version %d, %d hosts, %d groups, %d aliases.\n", version, stats.Hosts, stats.Groups, stats.Aliases)
		fmt.Fprintln(r.Out, "Dry run complete; the local configuration was not changed.")
		return nil
	}

	backup, err := installConfigAtomically(target, data, settings.Backup, settings.BackupRetention, r.Now())
	if err != nil {
		return err
	}
	if backup != "" {
		fmt.Fprintf(r.Out, "Backup created: %s\n", backup)
	}
	fmt.Fprintf(r.Out, "Updated %s to version %d (%d hosts, %d groups, %d aliases).\n", target, version, stats.Hosts, stats.Groups, stats.Aliases)

	newState := syncState{
		Server:    settings.Server,
		Profile:   settings.Profile,
		ETag:      etag,
		Version:   version,
		SHA256:    digest,
		SyncedAt:  r.Now().UTC(),
		LocalPath: target,
	}
	if err := writeSyncState(statePath, newState); err != nil {
		fmt.Fprintf(r.ErrOut, "warning: configuration installed but sync state could not be saved: %v\n", err)
	}
	return nil
}

func (r *SyncRunner) status(args []string) error {
	flags := flag.NewFlagSet("sshw sync status", flag.ContinueOnError)
	flags.SetOutput(r.ErrOut)
	configPath := flags.String("config", "", "sync settings path")
	if err := flags.Parse(args); err != nil {
		return err
	}
	settings, resolvedConfigPath, err := r.loadSettings(*configPath)
	if err != nil {
		return err
	}
	state, _ := readSyncState(syncStatePath(resolvedConfigPath))
	endpoint, err := syncEndpoint(settings)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodHead, endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+settings.Token)
	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("unable to reach config server: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("config server returned %s", resp.Status)
	}
	remoteVersion, _ := strconv.Atoi(resp.Header.Get("X-SSHW-Version"))
	fmt.Fprintf(r.Out, "Local version:  %d\nRemote version: %d\n", state.Version, remoteVersion)
	target, expandErr := homedir.Expand(settings.Target)
	localMatches := expandErr == nil &&
		state.Server == settings.Server &&
		state.Profile == settings.Profile &&
		state.LocalPath == target &&
		localConfigMatchesState(target, state)
	if state.Version == remoteVersion && state.ETag != "" && localMatches {
		fmt.Fprintln(r.Out, "Status: up to date")
	} else if state.Version == remoteVersion && state.ETag != "" && !localMatches {
		fmt.Fprintln(r.Out, "Status: local configuration changed; sync required")
	} else {
		fmt.Fprintln(r.Out, "Status: update available")
	}
	return nil
}

func (r *SyncRunner) loadSettings(path string) (SyncSettings, string, error) {
	var err error
	if path == "" {
		path, err = r.defaultSyncConfigPath()
		if err != nil {
			return SyncSettings{}, "", err
		}
	}
	path, err = homedir.Expand(path)
	if err != nil {
		return SyncSettings{}, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SyncSettings{}, "", fmt.Errorf("sync is not configured; run `sshw sync init`")
		}
		return SyncSettings{}, "", err
	}
	settings := SyncSettings{Backup: true, BackupRetention: 5, Profile: defaultSyncProfile}
	if err := yaml.UnmarshalStrict(data, &settings); err != nil {
		return SyncSettings{}, "", fmt.Errorf("invalid sync settings: %w", err)
	}
	if err := validateSyncSettings(&settings); err != nil {
		return SyncSettings{}, "", err
	}
	return settings, path, nil
}

func validateSyncSettings(settings *SyncSettings) error {
	if settings.Version != syncConfigVersion {
		return fmt.Errorf("unsupported sync settings version %d", settings.Version)
	}
	settings.Server = strings.TrimRight(strings.TrimSpace(settings.Server), "/")
	if settings.Server == "" {
		return fmt.Errorf("sync server is required")
	}
	parsed, err := url.Parse(settings.Server)
	if err != nil || parsed.Host == "" {
		return fmt.Errorf("invalid sync server URL")
	}
	if parsed.Scheme != "https" {
		host := parsed.Hostname()
		local := host == "localhost" || host == "127.0.0.1" || host == "::1"
		if parsed.Scheme != "http" || (!settings.AllowInsecure && !local) {
			return fmt.Errorf("sync server must use HTTPS (set allow-insecure only for trusted development servers)")
		}
	}
	if strings.TrimSpace(settings.Token) == "" {
		return fmt.Errorf("sync token is required")
	}
	if settings.Profile == "" {
		settings.Profile = defaultSyncProfile
	}
	if !syncProfilePattern.MatchString(settings.Profile) {
		return fmt.Errorf("profile may only contain letters, numbers, dots, underscores, and hyphens")
	}
	if strings.TrimSpace(settings.Target) == "" {
		return fmt.Errorf("sync target is required")
	}
	if settings.BackupRetention < 0 || settings.BackupRetention > 100 {
		return fmt.Errorf("backup-retention must be between 0 and 100")
	}
	return nil
}

func syncEndpoint(settings SyncSettings) (string, error) {
	if !syncProfilePattern.MatchString(settings.Profile) {
		return "", fmt.Errorf("invalid profile")
	}
	return settings.Server + "/api/v1/sync/" + settings.Profile, nil
}

func (r *SyncRunner) defaultSyncConfigPath() (string, error) {
	home, err := r.homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sshw-sync.yaml"), nil
}

func (r *SyncRunner) defaultTargetPath() (string, error) {
	home, err := r.homeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sshw"), nil
}

func (r *SyncRunner) homeDir() (string, error) {
	if r.HomeDir != "" {
		return r.HomeDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("find home directory: %w", err)
	}
	return home, nil
}

func syncStatePath(configPath string) string {
	ext := filepath.Ext(configPath)
	return strings.TrimSuffix(configPath, ext) + "-state.json"
}

func readSyncState(path string) (syncState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return syncState{}, err
	}
	var state syncState
	if err := json.Unmarshal(data, &state); err != nil {
		return syncState{}, err
	}
	return state, nil
}

func localConfigMatchesState(path string, state syncState) bool {
	if strings.TrimSpace(state.SHA256) == "" {
		return false
	}
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return false
	}
	return strings.EqualFold(hex.EncodeToString(hash.Sum(nil)), state.SHA256)
}

func writeSyncState(path string, state syncState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writePrivateFile(path, data)
}

func writePrivateFile(path string, data []byte) error {
	path, err := homedir.Expand(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return err
	}
	if _, err := io.Copy(temp, bytes.NewReader(data)); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	rollback := tempName + ".rollback"
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, rollback); err != nil {
			return err
		}
		if err := os.Rename(tempName, path); err != nil {
			if rollbackErr := os.Rename(rollback, path); rollbackErr != nil {
				return fmt.Errorf("replace %s: %v; rollback failed: %v", path, err, rollbackErr)
			}
			return err
		}
		_ = os.Remove(rollback)
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(tempName, path)
}

func installConfigAtomically(target string, data []byte, keepBackup bool, retention int, now time.Time) (string, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
		return "", fmt.Errorf("create config directory: %w", err)
	}
	temp, err := os.CreateTemp(filepath.Dir(target), "."+filepath.Base(target)+".sync-*")
	if err != nil {
		return "", fmt.Errorf("create temporary config: %w", err)
	}
	tempName := temp.Name()
	defer os.Remove(tempName)
	if err := temp.Chmod(0600); err != nil {
		temp.Close()
		return "", err
	}
	if _, err := temp.Write(data); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Sync(); err != nil {
		temp.Close()
		return "", err
	}
	if err := temp.Close(); err != nil {
		return "", err
	}

	var backup string
	if retention == 0 {
		keepBackup = false
	}
	if _, err := os.Stat(target); err == nil {
		suffix := now.Format("20060102-150405")
		if keepBackup {
			backup = uniqueBackupPath(target + ".backup-" + suffix)
		} else {
			backup = uniqueBackupPath(target + ".rollback-" + suffix)
		}
		if err := os.Rename(target, backup); err != nil {
			return "", fmt.Errorf("protect existing configuration: %w", err)
		}
		if err := os.Rename(tempName, target); err != nil {
			rollbackErr := os.Rename(backup, target)
			if rollbackErr != nil {
				return "", fmt.Errorf("install configuration: %v; rollback also failed: %v (original remains at %s)", err, rollbackErr, backup)
			}
			return "", fmt.Errorf("install configuration: %w", err)
		}
		if !keepBackup {
			_ = os.Remove(backup)
			backup = ""
		}
	} else if errors.Is(err, os.ErrNotExist) {
		if err := os.Rename(tempName, target); err != nil {
			return "", fmt.Errorf("install configuration: %w", err)
		}
	} else {
		return "", fmt.Errorf("inspect existing configuration: %w", err)
	}

	if keepBackup {
		pruneBackups(target, retention)
	}
	return backup, nil
}

func uniqueBackupPath(base string) string {
	if _, err := os.Stat(base); errors.Is(err, os.ErrNotExist) {
		return base
	}
	for i := 1; ; i++ {
		candidate := fmt.Sprintf("%s.%d", base, i)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
}

func pruneBackups(target string, retention int) {
	matches, err := filepath.Glob(target + ".backup-*")
	if err != nil || len(matches) <= retention {
		return
	}
	sort.Strings(matches)
	for _, path := range matches[:len(matches)-retention] {
		_ = os.Remove(path)
	}
}
