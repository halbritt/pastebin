package paste

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestValidateContent(t *testing.T) {
	tests := []struct {
		name    string
		content []byte
		max     int64
		wantErr error
	}{
		{name: "text", content: []byte("hello"), max: 10},
		{name: "unicode text", content: []byte("hello, \xcf\x80"), max: 20},
		{name: "whitespace allowed", content: []byte(" \t\r\n"), max: 10},
		{name: "exactly max bytes allowed", content: []byte("hello"), max: 5},
		{name: "empty rejected", content: nil, max: 10, wantErr: ErrEmpty},
		{name: "zero length rejected", content: []byte{}, max: 10, wantErr: ErrEmpty},
		{name: "too large", content: []byte("hello"), max: 4, wantErr: ErrTooLarge},
		{name: "too large checked before utf8", content: []byte{0xff, 0xff}, max: 1, wantErr: ErrTooLarge},
		{name: "invalid utf8", content: []byte{0xff}, max: 10, wantErr: ErrInvalidUTF8},
		{name: "nul byte rejected", content: []byte{'a', 0x00, 'b'}, max: 10, wantErr: ErrInvalidUTF8},
		{name: "escape byte rejected", content: []byte{'a', 0x1b, 'b'}, max: 10, wantErr: ErrInvalidUTF8},
		{name: "default max bytes enforced", content: bytes.Repeat([]byte("a"), int(DefaultMaxBytes)+1), max: 0, wantErr: ErrTooLarge},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateContent(tt.content, tt.max)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateContent() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContentDoesNotNormalizeContent(t *testing.T) {
	content := []byte(" leading\ttext\r\ntrailing spaces  ")
	want := append([]byte(nil), content...)

	if err := ValidateContent(content, 100); err != nil {
		t.Fatalf("ValidateContent() error = %v", err)
	}
	if !bytes.Equal(content, want) {
		t.Fatalf("ValidateContent() mutated content: got %q, want %q", content, want)
	}
}

func TestParseAllowedTTL(t *testing.T) {
	for value, want := range map[string]time.Duration{
		"":     DefaultTTL,
		" 1H ": time.Hour,
		"1h":   time.Hour,
		"1d":   24 * time.Hour,
		"7d":   DefaultTTL,
		"7D":   DefaultTTL,
		"30d":  MaxTTL,
	} {
		got, err := ParseAllowedTTL(value)
		if err != nil {
			t.Fatalf("ParseAllowedTTL(%q) error = %v", value, err)
		}
		if got != want {
			t.Fatalf("ParseAllowedTTL(%q) = %v, want %v", value, got, want)
		}
	}

	if _, err := ParseAllowedTTL("2d"); !errors.Is(err, ErrInvalidTTL) {
		t.Fatalf("ParseAllowedTTL(2d) error = %v, want ErrInvalidTTL", err)
	}
}

func TestValidateTTL(t *testing.T) {
	tests := []struct {
		name       string
		ttl        time.Duration
		defaultTTL time.Duration
		maxTTL     time.Duration
		want       time.Duration
		wantErr    error
	}{
		{name: "zero uses default", ttl: 0, defaultTTL: DefaultTTL, maxTTL: MaxTTL, want: DefaultTTL},
		{name: "zero uses package defaults", ttl: 0, defaultTTL: 0, maxTTL: 0, want: DefaultTTL},
		{name: "one hour allowed", ttl: time.Hour, defaultTTL: DefaultTTL, maxTTL: MaxTTL, want: time.Hour},
		{name: "max allowed", ttl: MaxTTL, defaultTTL: DefaultTTL, maxTTL: MaxTTL, want: MaxTTL},
		{name: "negative rejected", ttl: -time.Second, defaultTTL: DefaultTTL, maxTTL: MaxTTL, wantErr: ErrInvalidTTL},
		{name: "over max rejected", ttl: MaxTTL + time.Nanosecond, defaultTTL: DefaultTTL, maxTTL: MaxTTL, wantErr: ErrInvalidTTL},
		{name: "default over max rejected", ttl: 0, defaultTTL: MaxTTL + time.Hour, maxTTL: MaxTTL, wantErr: ErrInvalidTTL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidateTTL(tt.ttl, tt.defaultTTL, tt.maxTTL)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateTTL() error = %v, want %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ValidateTTL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNormalizeCode(t *testing.T) {
	if got := NormalizeCode("\tAbC2389 \r\n"); got != "abc2389" {
		t.Fatalf("NormalizeCode() = %q", got)
	}
}

func TestCodeAlphabetAvoidsVisuallyConfusingCharacters(t *testing.T) {
	for _, ch := range "ilo01ILO" {
		if strings.ContainsRune(codeAlphabet, ch) {
			t.Fatalf("code alphabet contains visually confusing character %q", ch)
		}
	}
}

func TestNewCode(t *testing.T) {
	code, err := newCode(bytes.NewReader([]byte{0, 1, 30, 31}), 4)
	if err != nil {
		t.Fatalf("newCode() error = %v", err)
	}
	want := string([]byte{codeAlphabet[0], codeAlphabet[1], codeAlphabet[30], codeAlphabet[0]})
	if code != want {
		t.Fatalf("newCode() = %q, want %q", code, want)
	}
	if !isNormalizedGeneratedCode(code) {
		t.Fatalf("newCode() produced invalid code %q", code)
	}
}

func TestNewCodeUsesDefaultSize(t *testing.T) {
	code, err := newCode(bytes.NewReader(bytes.Repeat([]byte{0}, DefaultCodeSize)), 0)
	if err != nil {
		t.Fatalf("newCode() error = %v", err)
	}
	if len(code) != DefaultCodeSize {
		t.Fatalf("newCode() length = %d, want %d", len(code), DefaultCodeSize)
	}
}

func TestNewCodeRejectsModuloBias(t *testing.T) {
	code, err := newCode(bytes.NewReader([]byte{255, 0, 1}), 2)
	if err != nil {
		t.Fatalf("newCode() error = %v", err)
	}
	want := string([]byte{codeAlphabet[0], codeAlphabet[1]})
	if code != want {
		t.Fatalf("newCode() = %q, want %q", code, want)
	}
}

func TestNewCodeErrors(t *testing.T) {
	if _, err := newCode(bytes.NewReader([]byte{255}), 1); !errors.Is(err, io.EOF) {
		t.Fatalf("newCode(exhausted) error = %v, want io.EOF", err)
	}
}

func isNormalizedGeneratedCode(code string) bool {
	if code == "" || code != NormalizeCode(code) {
		return false
	}
	for i := 0; i < len(code); i++ {
		if !strings.ContainsRune(codeAlphabet, rune(code[i])) {
			return false
		}
	}
	return true
}
