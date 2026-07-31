package sshw

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"time"

	pkgsftp "github.com/pkg/sftp"
)

const partialFileSuffix = ".sshw.part"

const posixRenameExtension = "posix-rename@openssh.com"

type remoteFileCommitter interface {
	HasExtension(string) (string, bool)
	PosixRename(string, string) error
	Rename(string, string) error
	Remove(string) error
	Stat(string) (os.FileInfo, error)
}

func (s *sftpSession) download(remoteValue, localValue string, force bool) error {
	return s.downloadWithResume(remoteValue, localValue, force, false)
}

func (s *sftpSession) reget(remoteValue, localValue string, force bool) error {
	return s.downloadWithResume(remoteValue, localValue, force, true)
}

func (s *sftpSession) downloadWithResume(
	remoteValue, localValue string,
	force, resume bool,
) error {
	remotePath := s.resolveRemote(remoteValue)
	localPath := localValue
	if localPath == "" {
		localPath = path.Base(remotePath)
	}
	localPath = s.resolveLocal(localPath)
	if info, statErr := os.Stat(localPath); statErr == nil && info.IsDir() {
		localPath = filepath.Join(localPath, path.Base(remotePath))
	}
	return s.downloadFile(remotePath, localPath, force, resume)
}

func (s *sftpSession) downloadFile(
	remotePath, localPath string,
	force, resume bool,
) error {
	remoteFile, err := s.client.Open(remotePath)
	if err != nil {
		return err
	}
	defer remoteFile.Close()

	remoteInfo, err := remoteFile.Stat()
	if err != nil {
		return err
	}
	if remoteInfo.IsDir() {
		return fmt.Errorf("%s is a directory", remotePath)
	}

	if exists, err := localPathExists(localPath); err != nil {
		return err
	} else if exists && !force {
		command := "get -f"
		if resume {
			command = "reget -f"
		}
		return fmt.Errorf("%s already exists; use %s to overwrite", localPath, command)
	}

	partialPath := filepath.Join(
		filepath.Dir(localPath),
		"."+filepath.Base(localPath)+partialFileSuffix,
	)
	metadataPath := partialPath + metadataFileSuffix
	var (
		localFile *os.File
		offset    int64
	)
	if resume {
		metadata, err := readLocalMetadata(metadataPath)
		if err != nil {
			return fmt.Errorf("read partial metadata %s: %w", metadataPath, err)
		}
		if err := metadata.validate(remoteFile, remoteInfo); err != nil {
			return fmt.Errorf("cannot resume download: %w", err)
		}
		partialInfo, err := os.Stat(partialPath)
		if err != nil {
			return fmt.Errorf("stat partial download %s: %w", partialPath, err)
		}
		offset = partialInfo.Size()
		if offset > remoteInfo.Size() {
			return fmt.Errorf(
				"partial download is larger than remote file: %d > %d",
				offset,
				remoteInfo.Size(),
			)
		}
		partialReader, err := os.Open(partialPath)
		if err != nil {
			return err
		}
		err = validatePartialPrefix(remoteFile, partialReader, offset)
		_ = partialReader.Close()
		if err != nil {
			return fmt.Errorf("cannot resume download: %w", err)
		}
		localFile, err = os.OpenFile(partialPath, os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		if _, err := localFile.Seek(offset, io.SeekStart); err != nil {
			_ = localFile.Close()
			return err
		}
		if _, err := remoteFile.Seek(offset, io.SeekStart); err != nil {
			_ = localFile.Close()
			return err
		}
	} else {
		if exists, err := localPathExists(partialPath); err != nil {
			return err
		} else if exists {
			return fmt.Errorf(
				"partial download exists at %s; use reget to resume",
				partialPath,
			)
		}
		if exists, err := localPathExists(metadataPath); err != nil {
			return err
		} else if exists {
			return fmt.Errorf(
				"partial metadata exists at %s; remove it or use reget",
				metadataPath,
			)
		}
		metadata, err := newTransferMetadata(remoteFile, remoteInfo)
		if err != nil {
			return fmt.Errorf("fingerprint remote file: %w", err)
		}
		localFile, err = os.OpenFile(
			partialPath,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
			0o600,
		)
		if err != nil {
			return err
		}
		if err := writeLocalMetadata(metadataPath, metadata); err != nil {
			_ = localFile.Close()
			_ = os.Remove(partialPath)
			return err
		}
	}
	defer localFile.Close()

	var (
		copySource io.Reader = remoteFile
		progress   *transferProgressReader
	)
	if s.directoryProgress != nil {
		copySource = s.directoryProgress.Reader(remoteFile)
	} else {
		progress = newTransferProgressReaderAt(
			remoteFile,
			s.output,
			"Downloading",
			remoteInfo.Size(),
			offset,
		)
		copySource = progress
	}
	written, err := io.Copy(localFile, copySource)
	if progress != nil {
		progress.Finish()
	}
	if err != nil {
		return fmt.Errorf(
			"download interrupted after %s; partial file preserved at %s; resume with reget: %w",
			formatByteCount(offset+written),
			partialPath,
			err,
		)
	}
	if err := localFile.Sync(); err != nil {
		return err
	}
	if err := localFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(partialPath, remoteInfo.Mode().Perm()); err != nil {
		return err
	}
	if err := os.Chtimes(partialPath, remoteInfo.ModTime(), remoteInfo.ModTime()); err != nil {
		return err
	}
	if err := replaceLocalFile(partialPath, localPath, force); err != nil {
		return err
	}
	if err := os.Remove(metadataPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("download completed, but remove metadata %s: %w", metadataPath, err)
	}

	if s.directoryProgress == nil {
		fmt.Fprintf(
			s.output,
			"Downloaded %s -> %s (%d bytes)\n",
			remotePath,
			localPath,
			offset+written,
		)
	}
	return nil
}

func (s *sftpSession) upload(localValue, remoteValue string, force bool) error {
	return s.uploadWithResume(localValue, remoteValue, force, false)
}

func (s *sftpSession) reput(localValue, remoteValue string, force bool) error {
	return s.uploadWithResume(localValue, remoteValue, force, true)
}

func (s *sftpSession) uploadWithResume(
	localValue, remoteValue string,
	force, resume bool,
) error {
	localPath := s.resolveLocal(localValue)
	localFile, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer localFile.Close()

	localInfo, err := localFile.Stat()
	if err != nil {
		return err
	}
	if localInfo.IsDir() {
		return fmt.Errorf("%s is a directory", localPath)
	}

	remotePath := remoteValue
	if remotePath == "" {
		remotePath = filepath.Base(localPath)
	}
	remotePath = s.resolveRemote(remotePath)
	if info, statErr := s.client.Stat(remotePath); statErr == nil && info.IsDir() {
		remotePath = path.Join(remotePath, filepath.Base(localPath))
	}
	return s.uploadFile(localFile, localInfo, localPath, remotePath, force, resume)
}

func (s *sftpSession) uploadFile(
	localFile *os.File,
	localInfo os.FileInfo,
	localPath, remotePath string,
	force, resume bool,
) error {
	if exists, err := s.remotePathExists(remotePath); err != nil {
		return err
	} else if exists && !force {
		command := "put -f"
		if resume {
			command = "reput -f"
		}
		return fmt.Errorf("%s already exists; use %s to overwrite", remotePath, command)
	}

	partialPath := remotePath + partialFileSuffix
	metadataPath := partialPath + metadataFileSuffix
	var (
		remoteFile *pkgsftp.File
		offset     int64
		err        error
	)
	if resume {
		metadata, err := s.readRemoteMetadata(metadataPath)
		if err != nil {
			return fmt.Errorf("read remote partial metadata %s: %w", metadataPath, err)
		}
		if err := metadata.validate(localFile, localInfo); err != nil {
			return fmt.Errorf("cannot resume upload: %w", err)
		}
		partialInfo, err := s.client.Stat(partialPath)
		if err != nil {
			return fmt.Errorf("stat partial upload %s: %w", partialPath, err)
		}
		offset = partialInfo.Size()
		if offset > localInfo.Size() {
			return fmt.Errorf(
				"partial upload is larger than local file: %d > %d",
				offset,
				localInfo.Size(),
			)
		}
		partialReader, err := s.client.Open(partialPath)
		if err != nil {
			return err
		}
		err = validatePartialPrefix(localFile, partialReader, offset)
		_ = partialReader.Close()
		if err != nil {
			return fmt.Errorf("cannot resume upload: %w", err)
		}
		remoteFile, err = s.client.OpenFile(partialPath, os.O_WRONLY)
		if err != nil {
			return err
		}
		if _, err := remoteFile.Seek(offset, io.SeekStart); err != nil {
			_ = remoteFile.Close()
			return err
		}
		if _, err := localFile.Seek(offset, io.SeekStart); err != nil {
			_ = remoteFile.Close()
			return err
		}
	} else {
		if exists, err := s.remotePathExists(partialPath); err != nil {
			return err
		} else if exists {
			return fmt.Errorf(
				"partial upload exists at %s; use reput to resume",
				partialPath,
			)
		}
		if exists, err := s.remotePathExists(metadataPath); err != nil {
			return err
		} else if exists {
			return fmt.Errorf(
				"partial metadata exists at %s; remove it or use reput",
				metadataPath,
			)
		}
		metadata, err := newTransferMetadata(localFile, localInfo)
		if err != nil {
			return fmt.Errorf("fingerprint local file: %w", err)
		}
		remoteFile, err = s.client.OpenFile(
			partialPath,
			os.O_WRONLY|os.O_CREATE|os.O_EXCL,
		)
		if err != nil {
			return err
		}
		if err := s.writeRemoteMetadata(metadataPath, metadata); err != nil {
			_ = remoteFile.Close()
			_ = s.client.Remove(partialPath)
			return err
		}
	}
	defer remoteFile.Close()

	var (
		copySource io.Reader = localFile
		progress   *transferProgressReader
	)
	if s.directoryProgress != nil {
		copySource = s.directoryProgress.Reader(localFile)
	} else {
		progress = newTransferProgressReaderAt(
			localFile,
			s.output,
			"Uploading",
			localInfo.Size(),
			offset,
		)
		copySource = progress
	}
	written, err := io.Copy(remoteFile, copySource)
	if progress != nil {
		progress.Finish()
	}
	if err != nil {
		return fmt.Errorf(
			"upload interrupted after %s; partial file preserved at %s; resume with reput: %w",
			formatByteCount(offset+written),
			partialPath,
			err,
		)
	}
	if err := remoteFile.Chmod(localInfo.Mode().Perm()); err != nil {
		return err
	}
	if err := remoteFile.Close(); err != nil {
		return err
	}
	if err := commitRemoteFile(
		s.client,
		partialPath,
		remotePath,
		force,
		newRemoteBackupPath(remotePath),
	); err != nil {
		return err
	}
	if err := s.client.Remove(metadataPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("upload completed, but remove metadata %s: %w", metadataPath, err)
	}

	if s.directoryProgress == nil {
		fmt.Fprintf(
			s.output,
			"Uploaded %s -> %s (%d bytes)\n",
			localPath,
			remotePath,
			offset+written,
		)
	}
	return nil
}

func localPathExists(value string) (bool, error) {
	_, err := os.Stat(value)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

func (s *sftpSession) remotePathExists(value string) (bool, error) {
	_, err := s.client.Stat(value)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func replaceLocalFile(partialPath, finalPath string, force bool) error {
	err := os.Rename(partialPath, finalPath)
	if err == nil {
		return nil
	}
	if !force {
		return err
	}
	if _, statErr := os.Stat(finalPath); errors.Is(statErr, os.ErrNotExist) {
		return err
	} else if statErr != nil {
		return statErr
	}

	backupPath := fmt.Sprintf("%s.sshw.backup.%d", finalPath, time.Now().UnixNano())
	if err := os.Rename(finalPath, backupPath); err != nil {
		return fmt.Errorf("preserve original local file %s: %w", finalPath, err)
	}
	if err := os.Rename(partialPath, finalPath); err != nil {
		if rollbackErr := os.Rename(backupPath, finalPath); rollbackErr != nil {
			return fmt.Errorf(
				"replace failed: %v; rollback failed: %v; original preserved at %s",
				err,
				rollbackErr,
				backupPath,
			)
		}
		return fmt.Errorf("replace failed: %w; original local file restored", err)
	}
	if err := os.Remove(backupPath); err != nil {
		return fmt.Errorf(
			"download completed, but could not remove original backup %s: %w",
			backupPath,
			err,
		)
	}
	return nil
}

func newRemoteBackupPath(finalPath string) string {
	return fmt.Sprintf("%s.sshw.backup.%d", finalPath, time.Now().UnixNano())
}

func commitRemoteFile(
	client remoteFileCommitter,
	partialPath, finalPath string,
	force bool,
	backupPath string,
) error {
	if !force {
		return client.Rename(partialPath, finalPath)
	}

	if _, supported := client.HasExtension(posixRenameExtension); supported {
		if err := client.PosixRename(partialPath, finalPath); err != nil {
			return fmt.Errorf("atomic overwrite %s: %w", finalPath, err)
		}
		return nil
	}

	_, err := client.Stat(finalPath)
	if os.IsNotExist(err) {
		return client.Rename(partialPath, finalPath)
	}
	if err != nil {
		return fmt.Errorf("check existing remote file %s: %w", finalPath, err)
	}

	if err := client.Rename(finalPath, backupPath); err != nil {
		return fmt.Errorf("preserve original file %s: %w", finalPath, err)
	}

	if err := client.Rename(partialPath, finalPath); err != nil {
		return rollbackRemoteFile(client, finalPath, backupPath, err)
	}

	if err := client.Remove(backupPath); err != nil {
		return fmt.Errorf(
			"uploaded %s, but could not remove the original backup at %s: %w",
			finalPath,
			backupPath,
			err,
		)
	}
	return nil
}

func rollbackRemoteFile(
	client remoteFileCommitter,
	finalPath, backupPath string,
	commitErr error,
) error {
	_, statErr := client.Stat(finalPath)
	switch {
	case statErr == nil:
		return fmt.Errorf(
			"commit status is uncertain after error %v; the original file is preserved at %s",
			commitErr,
			backupPath,
		)
	case !os.IsNotExist(statErr):
		return fmt.Errorf(
			"could not verify %s after error %v: %v; the original file is preserved at %s",
			finalPath,
			commitErr,
			statErr,
			backupPath,
		)
	}

	if err := client.Rename(backupPath, finalPath); err != nil {
		return fmt.Errorf(
			"commit failed: %v; rollback failed: %v; the original file is preserved at %s",
			commitErr,
			err,
			backupPath,
		)
	}
	return fmt.Errorf("commit failed: %w; the original file was restored", commitErr)
}
