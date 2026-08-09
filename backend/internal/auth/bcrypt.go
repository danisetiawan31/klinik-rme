package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

const CostFactor = 12

// Hash generates a bcrypt hash of the given password using cost factor 12.
func Hash(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), CostFactor)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// Verify checks whether the provided plaintext password matches the bcrypt hash.
// Returns (true, nil) if password matches hash.
// Returns (false, nil) if password does not match hash (bcrypt.ErrMismatchedHashAndPassword).
// Returns (false, err) for any other technical error (e.g. invalid hash format).
func Verify(password, hash string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
