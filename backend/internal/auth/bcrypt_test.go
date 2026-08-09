package auth_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/danisetiawan31/klinik-rme/internal/auth"
)

func TestBcrypt_HashAndVerify(t *testing.T) {
	password := "SecretP@ssw0rd!123"

	hash, err := auth.Hash(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)

	// Verify correct password returns (true, nil)
	match, err := auth.Verify(password, hash)
	require.NoError(t, err)
	assert.True(t, match)

	// Verify wrong password returns (false, nil) — NOT an error
	match, err = auth.Verify("WrongPassword!", hash)
	require.NoError(t, err)
	assert.False(t, match)

	// Verify malformed hash returns (false, error) — technical error
	match, err = auth.Verify(password, "invalid_hash_string")
	require.Error(t, err)
	assert.False(t, match)
}

func TestBcrypt_SaltUniqueness(t *testing.T) {
	password := "SamePassword123"

	hash1, err := auth.Hash(password)
	require.NoError(t, err)

	hash2, err := auth.Hash(password)
	require.NoError(t, err)

	// Bcrypt automatically generates a new salt for each hash operation
	assert.NotEqual(t, hash1, hash2)

	// Both hashes should still verify correctly against the same password
	match1, err := auth.Verify(password, hash1)
	require.NoError(t, err)
	assert.True(t, match1)

	match2, err := auth.Verify(password, hash2)
	require.NoError(t, err)
	assert.True(t, match2)
}
