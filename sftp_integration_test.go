package sshw

import (
	"bytes"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"

	pkgsftp "github.com/pkg/sftp"
)

func TestSFTPTransfersAgainstLocalServer(t *testing.T) {
	client, _ := newLocalSFTPClient(t)
	remoteDir := t.TempDir()
	localDir := t.TempDir()
	session := &sftpSession{
		client:    client,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}

	uploadContent := bytes.Repeat([]byte("upload-content\n"), 4096)
	if err := os.WriteFile(filepath.Join(localDir, "upload.bin"), uploadContent, 0o640); err != nil {
		t.Fatal(err)
	}
	if err := session.upload("upload.bin", "remote.bin", false); err != nil {
		t.Fatalf("upload() error = %v", err)
	}

	remoteContent, err := os.ReadFile(filepath.Join(remoteDir, "remote.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(remoteContent, uploadContent) {
		t.Fatal("uploaded content differs from local source")
	}

	if err := session.download("remote.bin", "download.bin", false); err != nil {
		t.Fatalf("download() error = %v", err)
	}
	downloadContent, err := os.ReadFile(filepath.Join(localDir, "download.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(downloadContent, uploadContent) {
		t.Fatal("downloaded content differs from remote source")
	}
}

func TestSFTPAtomicOverwriteAgainstLocalServer(t *testing.T) {
	client, _ := newLocalSFTPClient(t)
	if _, supported := client.HasExtension(posixRenameExtension); !supported {
		t.Fatal("local SFTP server does not advertise POSIX rename")
	}

	remoteDir := t.TempDir()
	localDir := t.TempDir()
	session := &sftpSession{
		client:    client,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}

	remotePath := filepath.Join(remoteDir, "target.txt")
	if err := os.WriteFile(remotePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "new.txt"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := session.upload("new.txt", "target.txt", true); err != nil {
		t.Fatalf("upload(force) error = %v", err)
	}
	content, err := os.ReadFile(remotePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "replacement" {
		t.Fatalf("remote content = %q, want replacement", content)
	}
}

func TestSFTPBackupOverwriteAgainstLocalServer(t *testing.T) {
	if err := pkgsftp.SetSFTPExtensions(
		"hardlink@openssh.com",
		"statvfs@openssh.com",
	); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = pkgsftp.SetSFTPExtensions(
			"hardlink@openssh.com",
			posixRenameExtension,
			"statvfs@openssh.com",
		)
	})

	client, _ := newLocalSFTPClient(t)
	if _, supported := client.HasExtension(posixRenameExtension); supported {
		t.Fatal("local SFTP server unexpectedly advertises POSIX rename")
	}

	remoteDir := t.TempDir()
	localDir := t.TempDir()
	session := &sftpSession{
		client:    client,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}

	remotePath := filepath.Join(remoteDir, "target.txt")
	if err := os.WriteFile(remotePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(localDir, "new.txt"), []byte("replacement"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := session.upload("new.txt", "target.txt", true); err != nil {
		t.Fatalf("upload(force fallback) error = %v", err)
	}
	content, err := os.ReadFile(remotePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "replacement" {
		t.Fatalf("remote content = %q, want replacement", content)
	}
	backups, err := filepath.Glob(remotePath + ".sshw.backup.*")
	if err != nil {
		t.Fatal(err)
	}
	if len(backups) != 0 {
		t.Fatalf("successful overwrite left backups: %#v", backups)
	}
}

func TestSFTPDisconnectDuringUploadPreservesOriginal(t *testing.T) {
	client, disconnect := newLocalSFTPClient(t)
	remoteDir := t.TempDir()
	localDir := t.TempDir()
	session := &sftpSession{
		client:    client,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}

	remotePath := filepath.Join(remoteDir, "target.bin")
	if err := os.WriteFile(remotePath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	source := bytes.Repeat([]byte("new-content"), 128*1024)
	if err := os.WriteFile(filepath.Join(localDir, "source.bin"), source, 0o600); err != nil {
		t.Fatal(err)
	}

	disconnect.afterBytes(64 * 1024)
	if err := session.upload("source.bin", "target.bin", true); err == nil {
		t.Fatal("upload() succeeded after the connection was interrupted")
	}

	content, err := os.ReadFile(remotePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("original remote file changed after disconnect: %q", content)
	}

	resumeClient, _ := newLocalSFTPClient(t)
	resumeSession := &sftpSession{
		client:    resumeClient,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}
	if err := resumeSession.reput("source.bin", "target.bin", true); err != nil {
		t.Fatalf("reput() error = %v", err)
	}
	content, err = os.ReadFile(remotePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, source) {
		t.Fatal("resumed upload differs from local source")
	}
}

func TestSFTPDisconnectAndResumeDownload(t *testing.T) {
	client, disconnect := newLocalSFTPClient(t)
	remoteDir := t.TempDir()
	localDir := t.TempDir()
	session := &sftpSession{
		client:    client,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}

	source := bytes.Repeat([]byte("remote-content"), 128*1024)
	if err := os.WriteFile(filepath.Join(remoteDir, "source.bin"), source, 0o600); err != nil {
		t.Fatal(err)
	}

	// Initial metadata fingerprinting reads the first and last 64 KiB.
	// Disconnect after that validation, while the actual file copy is running.
	disconnect.afterWrittenBytes(512 * 1024)
	if err := session.download("source.bin", "target.bin", false); err == nil {
		t.Fatal("download() succeeded after the connection was interrupted")
	}

	resumeClient, _ := newLocalSFTPClient(t)
	resumeSession := &sftpSession{
		client:    resumeClient,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}
	if err := resumeSession.reget("source.bin", "target.bin", false); err != nil {
		t.Fatalf("reget() error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(localDir, "target.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, source) {
		t.Fatal("resumed download differs from remote source")
	}
}

func TestSFTPRecursiveTransfersAgainstLocalServer(t *testing.T) {
	client, _ := newLocalSFTPClient(t)
	remoteDir := t.TempDir()
	localDir := t.TempDir()
	session := &sftpSession{
		client:    client,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}

	sourceRoot := filepath.Join(localDir, "source-tree")
	if err := os.MkdirAll(filepath.Join(sourceRoot, "nested", "empty"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "root.txt"), []byte("root"), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(sourceRoot, "nested", "child.txt"),
		[]byte("child"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}

	if err := session.uploadRecursive("source-tree", "remote-tree", false); err != nil {
		t.Fatalf("uploadRecursive() error = %v", err)
	}
	remoteChild, err := os.ReadFile(filepath.Join(remoteDir, "remote-tree", "nested", "child.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(remoteChild) != "child" {
		t.Fatalf("remote child content = %q", remoteChild)
	}

	if err := session.downloadRecursive("remote-tree", "downloaded-tree", false); err != nil {
		t.Fatalf("downloadRecursive() error = %v", err)
	}
	localChild, err := os.ReadFile(
		filepath.Join(localDir, "downloaded-tree", "nested", "child.txt"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if string(localChild) != "child" {
		t.Fatalf("downloaded child content = %q", localChild)
	}
	if info, err := os.Stat(
		filepath.Join(localDir, "downloaded-tree", "nested", "empty"),
	); err != nil || !info.IsDir() {
		t.Fatalf("empty directory was not preserved: info=%v err=%v", info, err)
	}

	if err := os.MkdirAll(filepath.Join(remoteDir, "source-tree"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := session.uploadRecursive("source-tree", "", true); err != nil {
		t.Fatalf("uploadRecursive(default destination) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(remoteDir, "source-tree", "root.txt")); err != nil {
		t.Fatal("default upload destination did not reuse the existing same-name directory")
	}
	if _, err := os.Stat(
		filepath.Join(remoteDir, "source-tree", "source-tree"),
	); !os.IsNotExist(err) {
		t.Fatal("default upload destination duplicated the source directory name")
	}

	if err := os.MkdirAll(filepath.Join(localDir, "remote-tree"), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := session.downloadRecursive("remote-tree", "", true); err != nil {
		t.Fatalf("downloadRecursive(default destination) error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(localDir, "remote-tree", "root.txt")); err != nil {
		t.Fatal("default download destination did not reuse the existing same-name directory")
	}
	if _, err := os.Stat(
		filepath.Join(localDir, "remote-tree", "remote-tree"),
	); !os.IsNotExist(err) {
		t.Fatal("default download destination duplicated the source directory name")
	}
}

func TestSFTPResumeRejectsChangedSource(t *testing.T) {
	client, _ := newLocalSFTPClient(t)
	remoteDir := t.TempDir()
	localDir := t.TempDir()
	session := &sftpSession{
		client:    client,
		output:    io.Discard,
		localDir:  localDir,
		remoteDir: remoteDir,
	}

	localPath := filepath.Join(localDir, "source.bin")
	if err := os.WriteFile(localPath, []byte("original-source"), 0o600); err != nil {
		t.Fatal(err)
	}
	localFile, err := os.Open(localPath)
	if err != nil {
		t.Fatal(err)
	}
	info, err := localFile.Stat()
	if err != nil {
		t.Fatal(err)
	}
	metadata, err := newTransferMetadata(localFile, info)
	_ = localFile.Close()
	if err != nil {
		t.Fatal(err)
	}

	partialPath := filepath.Join(remoteDir, "target.bin") + partialFileSuffix
	if err := os.WriteFile(partialPath, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := session.writeRemoteMetadata(partialPath+metadataFileSuffix, metadata); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(localPath, []byte("modified-source"), 0o600); err != nil {
		t.Fatal(err)
	}

	err = session.reput("source.bin", "target.bin", false)
	if err == nil || !bytes.Contains([]byte(err.Error()), []byte("source file changed")) {
		t.Fatalf("reput() error = %v", err)
	}
}

type disconnectingReadWriteCloser struct {
	net.Conn
	mu             sync.Mutex
	readRemaining  int64
	readActive     bool
	writeRemaining int64
	writeActive    bool
}

func (c *disconnectingReadWriteCloser) afterBytes(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readRemaining = count
	c.readActive = true
}

func (c *disconnectingReadWriteCloser) afterWrittenBytes(count int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeRemaining = count
	c.writeActive = true
}

func (c *disconnectingReadWriteCloser) Read(buffer []byte) (int, error) {
	n, err := c.Conn.Read(buffer)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.readActive {
		c.readRemaining -= int64(n)
		if c.readRemaining <= 0 {
			c.readActive = false
			_ = c.Conn.Close()
		}
	}
	return n, err
}

func (c *disconnectingReadWriteCloser) Write(buffer []byte) (int, error) {
	n, err := c.Conn.Write(buffer)
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.writeActive {
		c.writeRemaining -= int64(n)
		if c.writeRemaining <= 0 {
			c.writeActive = false
			_ = c.Conn.Close()
		}
	}
	return n, err
}

func newLocalSFTPClient(
	t *testing.T,
) (*pkgsftp.Client, *disconnectingReadWriteCloser) {
	t.Helper()

	serverConn, clientConn := net.Pipe()
	disconnect := &disconnectingReadWriteCloser{Conn: serverConn}
	server, err := pkgsftp.NewServer(disconnect)
	if err != nil {
		t.Fatal(err)
	}
	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Serve()
	}()

	client, err := pkgsftp.NewClientPipe(clientConn, clientConn)
	if err != nil {
		_ = server.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
		select {
		case <-serverDone:
		default:
		}
	})
	return client, disconnect
}
