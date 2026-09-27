// Package auth implements password and token primitives.
package auth

import (
	"crypto/pbkdf2"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

const (
	passwordIterations = 600000
	passwordKeyLength  = 32
)

func ValidatePassword(password string) error {
	length := len([]rune(password))
	if length < 12 || length > 128 {
		return errors.New("password must be between 12 and 128 characters")
	}
	return nil
}

func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("generate password salt: %w", err)
	}
	key, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyLength)
	if err != nil {
		return "", fmt.Errorf("derive password hash: %w", err)
	}
	return strings.Join([]string{"pbkdf2-sha256", fmt.Sprint(passwordIterations), base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key)}, "$"), nil
}

func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "pbkdf2-sha256" || parts[1] != fmt.Sprint(passwordIterations) {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil || len(salt) != 16 {
		return false
	}
	expected, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil || len(expected) != passwordKeyLength {
		return false
	}
	actual, err := pbkdf2.Key(sha256.New, password, salt, passwordIterations, passwordKeyLength)
	return err == nil && subtle.ConstantTimeCompare(actual, expected) == 1
}
