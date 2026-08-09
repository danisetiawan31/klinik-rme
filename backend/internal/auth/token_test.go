package auth_test

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
)

func TestGenerateToken_FormatAndEntropy(t *testing.T) {
	token, err := auth.GenerateToken()
	require.NoError(t, err)

	// Base64URL without padding should contain only [A-Za-z0-9_-]
	base64URLRegex := regexp.MustCompile(`^[A-Za-z0-9_-]+$`)
	assert.Regexp(t, base64URLRegex, token)

	// 16 bytes (128-bit) encoded as unpadded base64url has exact length 22
	assert.GreaterOrEqual(t, len(token), 22)
	assert.NotContains(t, token, "=")
	assert.NotContains(t, token, "+")
	assert.NotContains(t, token, "/")
}

func TestGenerateToken_NoCollisionsInSample(t *testing.T) {
	sampleSize := 1000
	generated := make(map[string]bool, sampleSize)

	for i := 0; i < sampleSize; i++ {
		token, err := auth.GenerateToken()
		require.NoError(t, err)

		assert.False(t, generated[token], "collision detected in 1000 generated tokens!")
		generated[token] = true
	}

	assert.Len(t, generated, sampleSize)
}

func TestHashToken_ConsistencyAndHexFormat(t *testing.T) {
	rawToken := "sample_raw_token_12345"

	hash1 := auth.HashToken(rawToken)
	hash2 := auth.HashToken(rawToken)

	// SHA256 hashing is deterministic (same input produces same output)
	assert.Equal(t, hash1, hash2)

	// SHA256 hex string output is exactly 64 hexadecimal characters
	hexRegex := regexp.MustCompile(`^[0-9a-f]{64}$`)
	assert.Regexp(t, hexRegex, hash1)

	// Different tokens produce different hashes
	differentHash := auth.HashToken("different_raw_token_67890")
	assert.NotEqual(t, hash1, differentHash)
}
