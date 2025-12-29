package utils

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

func GenerateSecureKey(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("key length must be positive")
	}

	key := make([]byte, length)
	if _, err := rand.Read(key); err != nil {
		return "", fmt.Errorf("failed to generate secure key: %w", err)
	}

	return hex.EncodeToString(key), nil
}

// generates a 32-byte (256-bit) secure key
func GenerateSecureKey32() (string, error) {
	return GenerateSecureKey(32)
}

// generates a 16-byte (128-bit) secure key
func GenerateSecureKey16() (string, error) {
	return GenerateSecureKey(16)
}
