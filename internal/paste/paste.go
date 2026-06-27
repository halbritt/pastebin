package paste

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	DefaultMaxBytes int64 = 1 << 20
	DefaultCodeSize       = 16
	codeAlphabet          = "abcdefghjkmnpqrstuvwxyz23456789"
)

var (
	DefaultTTL = 7 * 24 * time.Hour
	MaxTTL     = 30 * 24 * time.Hour

	AllowedTTLs = map[string]time.Duration{
		"1h":  time.Hour,
		"1d":  24 * time.Hour,
		"7d":  DefaultTTL,
		"30d": MaxTTL,
	}

	ErrNotFound    = errors.New("paste not found")
	ErrExpired     = errors.New("paste expired")
	ErrEmpty       = errors.New("paste is empty")
	ErrTooLarge    = errors.New("paste is too large")
	ErrInvalidUTF8 = errors.New("paste is not valid utf-8")
	ErrInvalidTTL  = errors.New("invalid expiration")
)

type Paste struct {
	Code      string
	Content   []byte
	CreatedAt time.Time
	ExpiresAt time.Time
	Size      int64
}

type CreateRequest struct {
	Content []byte
	TTL     time.Duration
	Now     time.Time
}

type Store interface {
	Create(ctx context.Context, req CreateRequest) (Paste, error)
	Get(ctx context.Context, code string, now time.Time) (Paste, error)
	CleanupExpired(ctx context.Context, now time.Time) (int64, error)
}

func ValidateContent(content []byte, maxBytes int64) error {
	if maxBytes <= 0 {
		maxBytes = DefaultMaxBytes
	}
	if len(content) == 0 {
		return ErrEmpty
	}
	if int64(len(content)) > maxBytes {
		return ErrTooLarge
	}
	if !utf8.Valid(content) {
		return ErrInvalidUTF8
	}
	for len(content) > 0 {
		r, size := utf8.DecodeRune(content)
		if disallowedTextControl(r) {
			return ErrInvalidUTF8
		}
		content = content[size:]
	}
	return nil
}

func ParseAllowedTTL(value string) (time.Duration, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return DefaultTTL, nil
	}
	ttl, ok := AllowedTTLs[value]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrInvalidTTL, value)
	}
	return ttl, nil
}

func ValidateTTL(ttl, defaultTTL, maxTTL time.Duration) (time.Duration, error) {
	if defaultTTL <= 0 {
		defaultTTL = DefaultTTL
	}
	if maxTTL <= 0 {
		maxTTL = MaxTTL
	}
	if ttl < 0 {
		return 0, ErrInvalidTTL
	}
	if ttl == 0 {
		ttl = defaultTTL
	}
	if ttl > maxTTL {
		return 0, ErrInvalidTTL
	}
	return ttl, nil
}

func NormalizeCode(code string) string {
	return strings.ToLower(strings.TrimSpace(code))
}

func NewCode() (string, error) {
	return newCode(rand.Reader, DefaultCodeSize)
}

func newCode(r io.Reader, size int) (string, error) {
	if size <= 0 {
		size = DefaultCodeSize
	}
	out := make([]byte, size)
	var buf [1]byte
	limit := len(codeAlphabet) * (256 / len(codeAlphabet))
	for n := 0; n < size; {
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return "", err
		}
		if int(buf[0]) >= limit {
			continue
		}
		out[n] = codeAlphabet[int(buf[0])%len(codeAlphabet)]
		n++
	}
	return string(out), nil
}

func disallowedTextControl(r rune) bool {
	switch r {
	case '\t', '\n', '\r':
		return false
	default:
		return unicode.IsControl(r)
	}
}
