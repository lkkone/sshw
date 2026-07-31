package sshw

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestSplitSFTPCommand(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    []string
		wantErr bool
	}{
		{
			name:  "plain arguments",
			input: "get remote.txt local.txt",
			want:  []string{"get", "remote.txt", "local.txt"},
		},
		{
			name:  "quoted paths",
			input: `put "local release.zip" '/remote/releases/release 1.zip'`,
			want:  []string{"put", "local release.zip", "/remote/releases/release 1.zip"},
		},
		{
			name:  "escaped whitespace",
			input: `cd release\ files`,
			want:  []string{"cd", "release files"},
		},
		{
			name:  "windows path",
			input: `lcd C:\Users\developer\Downloads`,
			want:  []string{"lcd", `C:\Users\developer\Downloads`},
		},
		{
			name:    "unterminated quote",
			input:   `get "remote file`,
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := splitSFTPCommand(test.input)
			if test.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatalf("splitSFTPCommand() error = %v", err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("splitSFTPCommand() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseTransferArgs(t *testing.T) {
	options, err := parseTransferArgs(
		[]string{"-r", "remote.txt", "-f", "local.txt"},
		true,
	)
	if err != nil {
		t.Fatalf("parseTransferArgs() error = %v", err)
	}
	if !options.Force {
		t.Fatal("parseTransferArgs() force = false, want true")
	}
	if !options.Recursive {
		t.Fatal("parseTransferArgs() recursive = false, want true")
	}
	if !reflect.DeepEqual(options.Paths, []string{"remote.txt", "local.txt"}) {
		t.Fatalf("parseTransferArgs() paths = %#v", options.Paths)
	}
	if _, err := parseTransferArgs(nil, true); err == nil {
		t.Fatal("parseTransferArgs(nil) expected an error")
	}
	if _, err := parseTransferArgs([]string{"-r", "remote.txt"}, false); err == nil {
		t.Fatal("parseTransferArgs(-r, resume) expected an error")
	}
}

func TestScannerLineReader(t *testing.T) {
	var output bytes.Buffer
	reader, err := newSFTPLineReader(strings.NewReader("pwd\nexit\n"), &output)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()

	first, err := reader.Readline()
	if err != nil {
		t.Fatal(err)
	}
	second, err := reader.Readline()
	if err != nil {
		t.Fatal(err)
	}
	if first != "pwd" || second != "exit" {
		t.Fatalf("commands = %q, %q", first, second)
	}
	if got := output.String(); got != "sftp> sftp> " {
		t.Fatalf("prompt output = %q", got)
	}
}

func TestFormatByteCount(t *testing.T) {
	tests := map[int64]string{
		0:           "0 B",
		1024:        "1.0 KiB",
		1024 * 1024: "1.0 MiB",
	}
	for value, want := range tests {
		if got := formatByteCount(value); got != want {
			t.Errorf("formatByteCount(%d) = %q, want %q", value, got, want)
		}
	}
}

func TestEmptyDirectoryProgress(t *testing.T) {
	var output bytes.Buffer
	progress := &directoryTransferProgress{
		output:    &output,
		label:     "Downloading directory",
		startedAt: time.Now(),
		enabled:   true,
	}
	progress.draw(true)
	if !strings.Contains(output.String(), "100% 0/0 files 0 B/0 B") {
		t.Fatalf("empty directory progress = %q", output.String())
	}
}

func TestSFTPSessionResolvesPaths(t *testing.T) {
	session := &sftpSession{
		remoteDir: "/srv/releases",
		localDir:  filepath.Join("tmp", "downloads"),
	}

	if got := session.resolveRemote("../logs/app.log"); got != "/srv/logs/app.log" {
		t.Fatalf("resolveRemote() = %q", got)
	}
	if got := session.resolveRemote("/var/log/app.log"); got != "/var/log/app.log" {
		t.Fatalf("resolveRemote() absolute = %q", got)
	}

	wantLocal := filepath.Clean(filepath.Join("tmp", "downloads", "app.log"))
	if got := session.resolveLocal("app.log"); got != wantLocal {
		t.Fatalf("resolveLocal() = %q, want %q", got, wantLocal)
	}
}

func TestReplaceLocalFile(t *testing.T) {
	dir := t.TempDir()
	finalPath := filepath.Join(dir, "download.txt")
	partialPath := filepath.Join(dir, "download.part")

	if err := os.WriteFile(partialPath, []byte("new"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := replaceLocalFile(partialPath, finalPath, false); err != nil {
		t.Fatalf("replaceLocalFile() error = %v", err)
	}

	if err := os.WriteFile(partialPath, []byte("newer"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := replaceLocalFile(partialPath, finalPath, true); err != nil {
		t.Fatalf("replaceLocalFile(force) error = %v", err)
	}

	content, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "newer" {
		t.Fatalf("final content = %q", content)
	}
}

func TestCommitRemoteFileUsesPosixRenameWhenSupported(t *testing.T) {
	client := newFakeRemoteCommitter("target", "partial")
	client.hasPosixRename = true
	client.posixRenameErr = errors.New("connection lost")

	err := commitRemoteFile(client, "partial", "target", true, "backup")
	if err == nil || !strings.Contains(err.Error(), "atomic overwrite") {
		t.Fatalf("commitRemoteFile() error = %v", err)
	}

	want := []string{
		"extension:" + posixRenameExtension,
		"posix-rename:partial->target",
	}
	if !reflect.DeepEqual(client.operations, want) {
		t.Fatalf("operations = %#v, want %#v", client.operations, want)
	}
	if !client.files["target"] {
		t.Fatal("original target was modified after PosixRename failed")
	}
}

func TestCommitRemoteFileBacksUpOriginalWithoutPosixRename(t *testing.T) {
	client := newFakeRemoteCommitter("target", "partial")

	if err := commitRemoteFile(client, "partial", "target", true, "backup"); err != nil {
		t.Fatalf("commitRemoteFile() error = %v", err)
	}

	want := []string{
		"extension:" + posixRenameExtension,
		"stat:target",
		"rename:target->backup",
		"rename:partial->target",
		"remove:backup",
	}
	if !reflect.DeepEqual(client.operations, want) {
		t.Fatalf("operations = %#v, want %#v", client.operations, want)
	}
	if !client.files["target"] || client.files["backup"] {
		t.Fatalf("unexpected remote files after commit: %#v", client.files)
	}
}

func TestCommitRemoteFileRestoresOriginalWhenCommitFails(t *testing.T) {
	client := newFakeRemoteCommitter("target", "partial")
	client.renameErrors["partial->target"] = errors.New("permission denied")

	err := commitRemoteFile(client, "partial", "target", true, "backup")
	if err == nil || !strings.Contains(err.Error(), "original file was restored") {
		t.Fatalf("commitRemoteFile() error = %v", err)
	}
	if !client.files["target"] || client.files["backup"] {
		t.Fatalf("original file was not restored: %#v", client.files)
	}
}

func TestCommitRemoteFilePreservesBackupWhenStateCannotBeVerified(t *testing.T) {
	client := newFakeRemoteCommitter("target", "partial")
	client.renameErrors["partial->target"] = errors.New("connection lost")
	client.statErrors["target"] = []error{nil, errors.New("connection lost")}

	err := commitRemoteFile(client, "partial", "target", true, "backup")
	if err == nil || !strings.Contains(err.Error(), "original file is preserved at backup") {
		t.Fatalf("commitRemoteFile() error = %v", err)
	}
	if !client.files["backup"] {
		t.Fatalf("original backup was not preserved: %#v", client.files)
	}
}

type fakeRemoteCommitter struct {
	files          map[string]bool
	hasPosixRename bool
	posixRenameErr error
	renameErrors   map[string]error
	removeErrors   map[string]error
	statErrors     map[string][]error
	operations     []string
}

func newFakeRemoteCommitter(files ...string) *fakeRemoteCommitter {
	client := &fakeRemoteCommitter{
		files:        make(map[string]bool),
		renameErrors: make(map[string]error),
		removeErrors: make(map[string]error),
		statErrors:   make(map[string][]error),
	}
	for _, file := range files {
		client.files[file] = true
	}
	return client
}

func (c *fakeRemoteCommitter) HasExtension(name string) (string, bool) {
	c.operations = append(c.operations, "extension:"+name)
	if c.hasPosixRename {
		return "1", true
	}
	return "", false
}

func (c *fakeRemoteCommitter) PosixRename(oldPath, newPath string) error {
	c.operations = append(c.operations, "posix-rename:"+oldPath+"->"+newPath)
	if c.posixRenameErr != nil {
		return c.posixRenameErr
	}
	delete(c.files, oldPath)
	c.files[newPath] = true
	return nil
}

func (c *fakeRemoteCommitter) Rename(oldPath, newPath string) error {
	c.operations = append(c.operations, "rename:"+oldPath+"->"+newPath)
	if err := c.renameErrors[oldPath+"->"+newPath]; err != nil {
		return err
	}
	if !c.files[oldPath] {
		return &os.PathError{Op: "rename", Path: oldPath, Err: os.ErrNotExist}
	}
	delete(c.files, oldPath)
	c.files[newPath] = true
	return nil
}

func (c *fakeRemoteCommitter) Remove(value string) error {
	c.operations = append(c.operations, "remove:"+value)
	if err := c.removeErrors[value]; err != nil {
		return err
	}
	if !c.files[value] {
		return &os.PathError{Op: "remove", Path: value, Err: os.ErrNotExist}
	}
	delete(c.files, value)
	return nil
}

func (c *fakeRemoteCommitter) Stat(value string) (os.FileInfo, error) {
	c.operations = append(c.operations, "stat:"+value)
	if queued := c.statErrors[value]; len(queued) > 0 {
		err := queued[0]
		c.statErrors[value] = queued[1:]
		if err != nil {
			return nil, err
		}
	}
	if !c.files[value] {
		return nil, &os.PathError{Op: "stat", Path: value, Err: os.ErrNotExist}
	}
	return fakeFileInfo{name: value}, nil
}

type fakeFileInfo struct {
	name string
}

func (f fakeFileInfo) Name() string     { return f.name }
func (fakeFileInfo) Size() int64        { return 0 }
func (fakeFileInfo) Mode() os.FileMode  { return 0o600 }
func (fakeFileInfo) ModTime() time.Time { return time.Time{} }
func (fakeFileInfo) IsDir() bool        { return false }
func (fakeFileInfo) Sys() interface{}   { return nil }
