package sshw

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type remoteWalkEntry struct {
	path string
	info os.FileInfo
}

type localWalkEntry struct {
	path string
	info os.FileInfo
}

func (s *sftpSession) downloadRecursive(
	remoteValue, localValue string,
	force bool,
) error {
	remoteRoot := s.resolveRemote(remoteValue)
	rootInfo, err := s.client.Lstat(remoteRoot)
	if err != nil {
		return err
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("%s is not a directory; remove -r to download a file", remoteRoot)
	}

	destinationProvided := localValue != ""
	localRoot := localValue
	if localRoot == "" {
		localRoot = path.Base(remoteRoot)
	}
	localRoot = s.resolveLocal(localRoot)
	if info, statErr := os.Stat(localRoot); statErr == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", localRoot)
		}
		if destinationProvided {
			localRoot = filepath.Join(localRoot, path.Base(remoteRoot))
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	fmt.Fprintf(s.output, "Scanning remote directory %s...\n", remoteRoot)
	entries, totalBytes, err := s.collectRemoteTree(remoteRoot)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(localRoot, rootInfo.Mode().Perm()); err != nil {
		return err
	}

	var (
		transferred int64
		skipped     int
	)
	fileCount := countRemoteFiles(entries)
	progress := newDirectoryTransferProgress(
		s.output,
		"Downloading directory",
		totalBytes,
		fileCount,
	)
	s.directoryProgress = progress
	defer func() {
		if s.directoryProgress == progress {
			progress.Finish()
			s.directoryProgress = nil
		}
	}()
	for _, entry := range entries {
		relative := remoteRelative(remoteRoot, entry.path)
		if relative == "." {
			continue
		}
		localPath := filepath.Join(localRoot, filepath.FromSlash(relative))
		switch {
		case entry.info.IsDir():
			if err := os.MkdirAll(localPath, entry.info.Mode().Perm()); err != nil {
				return err
			}
		case entry.info.Mode()&os.ModeSymlink != 0:
			skipped++
		case entry.info.Mode().IsRegular():
			if err := s.downloadFile(entry.path, localPath, force, false); err != nil {
				return err
			}
			transferred += entry.info.Size()
			progress.FileDone()
		default:
			skipped++
		}
	}
	progress.Finish()
	s.directoryProgress = nil
	fmt.Fprintf(
		s.output,
		"Downloaded directory %s -> %s (%d files, %s, %d skipped)\n",
		remoteRoot,
		localRoot,
		fileCount,
		formatByteCount(minInt64(transferred, totalBytes)),
		skipped,
	)
	return nil
}

func (s *sftpSession) uploadRecursive(
	localValue, remoteValue string,
	force bool,
) error {
	localRoot := s.resolveLocal(localValue)
	rootInfo, err := os.Lstat(localRoot)
	if err != nil {
		return err
	}
	if !rootInfo.IsDir() {
		return fmt.Errorf("%s is not a directory; remove -r to upload a file", localRoot)
	}

	destinationProvided := remoteValue != ""
	remoteRoot := remoteValue
	if remoteRoot == "" {
		remoteRoot = filepath.Base(localRoot)
	}
	remoteRoot = s.resolveRemote(remoteRoot)
	if info, statErr := s.client.Stat(remoteRoot); statErr == nil {
		if !info.IsDir() {
			return fmt.Errorf("%s is not a directory", remoteRoot)
		}
		if destinationProvided {
			remoteRoot = path.Join(remoteRoot, filepath.Base(localRoot))
		}
	} else if !os.IsNotExist(statErr) {
		return statErr
	}
	fmt.Fprintf(s.output, "Scanning local directory %s...\n", localRoot)
	entries, totalBytes, err := collectLocalTree(localRoot)
	if err != nil {
		return err
	}
	if err := s.client.MkdirAll(remoteRoot); err != nil {
		return err
	}

	var (
		transferred int64
		skipped     int
	)
	fileCount := countLocalFiles(entries)
	progress := newDirectoryTransferProgress(
		s.output,
		"Uploading directory",
		totalBytes,
		fileCount,
	)
	s.directoryProgress = progress
	defer func() {
		if s.directoryProgress == progress {
			progress.Finish()
			s.directoryProgress = nil
		}
	}()
	for _, entry := range entries {
		relative, err := filepath.Rel(localRoot, entry.path)
		if err != nil {
			return err
		}
		if relative == "." {
			continue
		}
		remotePath := path.Join(remoteRoot, filepath.ToSlash(relative))
		switch {
		case entry.info.IsDir():
			if err := s.client.MkdirAll(remotePath); err != nil {
				return err
			}
			if err := s.client.Chmod(remotePath, entry.info.Mode().Perm()); err != nil {
				return err
			}
		case entry.info.Mode()&os.ModeSymlink != 0:
			skipped++
		case entry.info.Mode().IsRegular():
			localFile, err := os.Open(entry.path)
			if err != nil {
				return err
			}
			err = s.uploadFile(
				localFile,
				entry.info,
				entry.path,
				remotePath,
				force,
				false,
			)
			_ = localFile.Close()
			if err != nil {
				return err
			}
			transferred += entry.info.Size()
			progress.FileDone()
		default:
			skipped++
		}
	}
	progress.Finish()
	s.directoryProgress = nil
	fmt.Fprintf(
		s.output,
		"Uploaded directory %s -> %s (%d files, %s, %d skipped)\n",
		localRoot,
		remoteRoot,
		fileCount,
		formatByteCount(minInt64(transferred, totalBytes)),
		skipped,
	)
	return nil
}

func (s *sftpSession) collectRemoteTree(
	root string,
) ([]remoteWalkEntry, int64, error) {
	var (
		entries []remoteWalkEntry
		total   int64
	)
	walker := s.client.Walk(root)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return nil, 0, err
		}
		info := walker.Stat()
		entries = append(entries, remoteWalkEntry{path: walker.Path(), info: info})
		if info.Mode().IsRegular() {
			total += info.Size()
		}
	}
	return entries, total, nil
}

func collectLocalTree(root string) ([]localWalkEntry, int64, error) {
	var (
		entries []localWalkEntry
		total   int64
	)
	err := filepath.WalkDir(root, func(value string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		entries = append(entries, localWalkEntry{path: value, info: info})
		if info.Mode().IsRegular() {
			total += info.Size()
		}
		return nil
	})
	return entries, total, err
}

func countRemoteFiles(entries []remoteWalkEntry) int {
	count := 0
	for _, entry := range entries {
		if entry.info.Mode().IsRegular() {
			count++
		}
	}
	return count
}

func countLocalFiles(entries []localWalkEntry) int {
	count := 0
	for _, entry := range entries {
		if entry.info.Mode().IsRegular() {
			count++
		}
	}
	return count
}

func remoteRelative(root, value string) string {
	root = path.Clean(root)
	value = path.Clean(value)
	if value == root {
		return "."
	}
	return strings.TrimPrefix(value, strings.TrimSuffix(root, "/")+"/")
}
