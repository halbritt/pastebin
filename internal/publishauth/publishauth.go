package publishauth

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"os"
	"strings"
)

func ReadTokenFile(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", nil
	}
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read publish token: %w", err)
	}
	token := strings.TrimSpace(string(content))
	if err := ValidateToken(token); err != nil {
		return "", err
	}
	return token, nil
}

func ValidateToken(token string) error {
	if len(token) >= 32 {
		return nil
	}
	if len(token) == 22 {
		decoded, err := base64.RawURLEncoding.DecodeString(token)
		if err == nil && len(decoded) == 16 {
			return nil
		}
	}
	words := strings.Split(token, "-")
	if len(words) == 3 {
		valid := true
		for _, word := range words {
			if len(word) < 3 || len(word) > 9 {
				valid = false
				break
			}
			for _, letter := range word {
				if letter < 'a' || letter > 'z' {
					valid = false
					break
				}
			}
		}
		if valid {
			return nil
		}
	}
	return errors.New("publish token must be three lowercase words, 22 base64url characters, or at least 32 characters")
}

func MatchesBearer(token, authorization string) bool {
	scheme, credentials, ok := strings.Cut(strings.TrimSpace(authorization), " ")
	if !ok || !strings.EqualFold(scheme, "Bearer") {
		return false
	}
	expected := sha256.Sum256([]byte(token))
	provided := sha256.Sum256([]byte(strings.TrimSpace(credentials)))
	return subtle.ConstantTimeCompare(expected[:], provided[:]) == 1
}
