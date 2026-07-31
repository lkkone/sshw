package sshw

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	transferMetadataVersion = 1
	fingerprintChunkSize    = 64 * 1024
	metadataFileSuffix      = ".meta"
)

type transferMetadata struct {
	Version     int    `json:"version"`
	Size        int64  `json:"size"`
	ModTimeNano int64  `json:"mod_time_nano"`
	Fingerprint string `json:"fingerprint"`
}

type readerAt interface {
	ReadAt([]byte, int64) (int, error)
}

func newTransferMetadata(
	source readerAt,
	info os.FileInfo,
) (transferMetadata, error) {
	fingerprint, err := sourceFingerprint(source, info.Size())
	if err != nil {
		return transferMetadata{}, err
	}
	return transferMetadata{
		Version:     transferMetadataVersion,
		Size:        info.Size(),
		ModTimeNano: info.ModTime().UnixNano(),
		Fingerprint: fingerprint,
	}, nil
}

func (m transferMetadata) validate(
	source readerAt,
	info os.FileInfo,
) error {
	if m.Version != transferMetadataVersion {
		return fmt.Errorf("unsupported partial metadata version %d", m.Version)
	}
	if m.Size != info.Size() || m.ModTimeNano != info.ModTime().UnixNano() {
		return errors.New("source file changed since the interrupted transfer")
	}
	fingerprint, err := sourceFingerprint(source, info.Size())
	if err != nil {
		return err
	}
	if fingerprint != m.Fingerprint {
		return errors.New("source file content changed since the interrupted transfer")
	}
	return nil
}

func sourceFingerprint(source readerAt, size int64) (string, error) {
	hash := sha256.New()
	if _, err := fmt.Fprintf(hash, "%d:", size); err != nil {
		return "", err
	}
	if size == 0 {
		return fmt.Sprintf("%x", hash.Sum(nil)), nil
	}

	firstSize := minInt64(size, fingerprintChunkSize)
	if err := hashRange(hash, source, 0, firstSize); err != nil {
		return "", err
	}
	if size > firstSize {
		lastSize := minInt64(size-firstSize, fingerprintChunkSize)
		if err := hashRange(hash, source, size-lastSize, lastSize); err != nil {
			return "", err
		}
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func validatePartialPrefix(
	source, partial readerAt,
	size int64,
) error {
	sourceValue, err := sourceFingerprint(source, size)
	if err != nil {
		return fmt.Errorf("fingerprint source prefix: %w", err)
	}
	partialValue, err := sourceFingerprint(partial, size)
	if err != nil {
		return fmt.Errorf("fingerprint partial file: %w", err)
	}
	if sourceValue != partialValue {
		return errors.New("partial file does not match the source file")
	}
	return nil
}

func hashRange(
	destination io.Writer,
	source readerAt,
	offset, size int64,
) error {
	buffer := make([]byte, size)
	n, err := source.ReadAt(buffer, offset)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if int64(n) != size {
		return io.ErrUnexpectedEOF
	}
	_, err = destination.Write(buffer)
	return err
}

func minInt64(left, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

func writeLocalMetadata(value string, metadata transferMetadata) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	return os.WriteFile(value, data, 0o600)
}

func readLocalMetadata(value string) (transferMetadata, error) {
	data, err := os.ReadFile(value)
	if err != nil {
		return transferMetadata{}, err
	}
	var metadata transferMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return transferMetadata{}, err
	}
	return metadata, nil
}

func (s *sftpSession) writeRemoteMetadata(
	value string,
	metadata transferMetadata,
) error {
	data, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	file, err := s.client.OpenFile(value, os.O_WRONLY|os.O_CREATE|os.O_TRUNC)
	if err != nil {
		return err
	}
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func (s *sftpSession) readRemoteMetadata(
	value string,
) (transferMetadata, error) {
	file, err := s.client.Open(value)
	if err != nil {
		return transferMetadata{}, err
	}
	defer file.Close()
	data, err := io.ReadAll(file)
	if err != nil {
		return transferMetadata{}, err
	}
	var metadata transferMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return transferMetadata{}, err
	}
	return metadata, nil
}
