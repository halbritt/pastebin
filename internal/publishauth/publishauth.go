package publishauth

import (
	"crypto/sha256"
	"crypto/subtle"
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
	if len(token) < 32 {
		return errors.New("publish token must contain at least 32 characters")
	}
	return nil
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
