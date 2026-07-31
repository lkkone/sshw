package configserver

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type vault struct {
	aead cipher.AEAD
}

func newVault(secret string) (*vault, error) {
	secret = strings.TrimSpace(secret)
	if secret == "" {
		return nil, fmt.Errorf("master key is required")
	}
	var key []byte
	if decoded, err := base64.StdEncoding.DecodeString(secret); err == nil && len(decoded) == 32 {
		key = decoded
	} else {
		sum := sha256.Sum256([]byte(secret))
		key = sum[:]
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &vault{aead: aead}, nil
}

func (v *vault) encrypt(plaintext []byte) ([]byte, error) {
	nonce := make([]byte, v.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return append(nonce, v.aead.Seal(nil, nonce, plaintext, nil)...), nil
}

func (v *vault) decrypt(ciphertext []byte) ([]byte, error) {
	nonceSize := v.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("encrypted value is truncated")
	}
	plaintext, err := v.aead.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt stored value: %w", err)
	}
	return plaintext, nil
}
