package utils

import (
	"crypto/rand"
	"math/big"
	"strings"
	"time"
)

const (
	usernameChars = "abcdefghijklmnopqrstuvwxyz0123456789"
	passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%&*"
)

// RandomString generates a cryptographically random string of the given length.
func RandomString(length int, charset string) string {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))
	for i := range result {
		n, err := rand.Int(rand.Reader, charsetLen)
		if err != nil || n == nil {
			// Fallback in case crypto/rand fails
			result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
			continue
		}
		result[i] = charset[n.Int64()]
	}
	return string(result)
}

// RandomUsername generates a random username.
func RandomUsername() string {
	return RandomString(10, usernameChars)
}

// RandomPassword generates a random password.
func RandomPassword() string {
	return RandomString(12, passwordChars)
}

// IsEmailValid basic check.
func IsEmailValid(email string) bool {
	return strings.Contains(email, "@") && len(email) > 3
}
