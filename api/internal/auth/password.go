package auth

import "golang.org/x/crypto/bcrypt"

// MinPasswordLength is the shortest password accepted at sign-up.
const MinPasswordLength = 8

// HashPassword returns a bcrypt hash of the password.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches the bcrypt hash.
func CheckPassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword checks the password policy and returns a human readable
// problem, or "" if the password is fine.
func ValidatePassword(password string) string {
	switch {
	case len(password) < MinPasswordLength:
		return "must be at least 8 characters"
	case len(password) > 72:
		return "must be at most 72 bytes"
	}
	return ""
}
