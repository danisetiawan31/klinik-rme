package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

const TokenEntropyBytes = 16 // 128-bit entropy

// GenerateToken generates a cryptographically secure random token with 128-bit entropy,
// encoded as unpadded base64url string.
func GenerateToken() (string, error) {
	bytes := make([]byte, TokenEntropyBytes)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

// HashToken computes the SHA256 hash of a raw token string and returns it as hex string.
// This pattern is used consistently across sessions.id_hash, password_tokens.token_hash,
// and klinik.display_token_hash.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
