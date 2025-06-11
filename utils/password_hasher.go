package utils

import (
	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes the given password using bcrypt.
func HashPassword(password string) (string, error) {
	// GeneSubscriptionFromPassword returns a bcrypt hashed password
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// CheckPasswordHash compares a plaintext password with a hashed password to check if they match.
func CheckPasswordHash(password, hash string) bool {
	// CompareHashAndPassword returns nil if the password and hash match
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
