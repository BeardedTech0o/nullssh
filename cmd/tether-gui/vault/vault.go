// Package vault protects saved connection passwords behind a master
// password. The master password itself is never stored: instead, a random
// salt and a "check" value (a fixed plaintext encrypted with the key
// derived from the master password) are persisted, and a candidate password
// is verified by deriving a key and attempting to decrypt the check value
// with it.
package vault

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	// CheckPlaintext is encrypted with the derived key and stored
	// alongside the salt; a master password is correct if and only if it
	// decrypts back to this value.
	CheckPlaintext = "tether-vault-check"

	pbkdf2Iterations = 600_000
	KeyLength        = 32 // AES-256
	SaltLength       = 16
)

// NewSalt generates a random salt suitable for DeriveKey.
func NewSalt() ([]byte, error) {
	return randomBytes(SaltLength)
}

func randomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}
	return b, nil
}

// DeriveKey derives a symmetric encryption key from a password and salt.
func DeriveKey(password string, salt []byte) ([]byte, error) {
	key, err := pbkdf2.Key(sha256.New, password, salt, pbkdf2Iterations, KeyLength)
	if err != nil {
		return nil, fmt.Errorf("derive key: %w", err)
	}
	return key, nil
}

// EncryptString encrypts plaintext with key (AES-256-GCM) and returns the
// base64-encoded nonce+ciphertext.
func EncryptString(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce, err := randomBytes(gcm.NonceSize())
	if err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptString reverses EncryptString. It fails if key is wrong or encoded
// has been tampered with (AES-GCM is authenticated).
func DecryptString(key []byte, encoded string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(data) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ciphertext := data[:gcm.NonceSize()], data[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: %w (wrong master password?)", err)
	}
	return string(plaintext), nil
}
