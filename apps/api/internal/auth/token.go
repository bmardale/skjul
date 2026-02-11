package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const tokenSize = 32

func GenerateSessionToken() (raw string, hash string, err error) {
	b := make([]byte, tokenSize)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("generate token: %w", err)
	}

	raw = base64.RawURLEncoding.EncodeToString(b)
	hash = HashSessionToken(raw)
	return raw, hash, nil
}

func HashSessionToken(raw string) string {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return ""
	}
	sum := sha256.Sum256(b)
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
