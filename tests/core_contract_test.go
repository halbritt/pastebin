package tests

import (
	"bytes"
	"errors"
	"testing"
	"time"

	"pastebin/internal/paste"
)

func TestCorePasteContentContractRejectsOnlyUnacceptableText(t *testing.T) {
	valid := []byte("tabs\ttrailing spaces  \r\nno final newline")
	before := append([]byte(nil), valid...)

	if err := paste.ValidateContent(valid, paste.DefaultMaxBytes); err != nil {
		t.Fatalf("ValidateContent(valid text) error = %v", err)
	}
	if !bytes.Equal(valid, before) {
		t.Fatalf("ValidateContent changed content bytes: got %q, want %q", valid, before)
	}

	tests := []struct {
		name    string
		content []byte
		wantErr error
	}{
		{name: "empty paste", content: nil, wantErr: paste.ErrEmpty},
		{name: "invalid UTF-8 paste", content: []byte{0xff}, wantErr: paste.ErrInvalidUTF8},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := paste.ValidateContent(tt.content, paste.DefaultMaxBytes)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ValidateContent() error = %v, want %v", err, tt.wantErr)
			}
		})
	}
}

func TestCoreExpirationContractMatchesAcceptedTTLPolicy(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  time.Duration
	}{
		{name: "default", value: "", want: 7 * 24 * time.Hour},
		{name: "one hour", value: "1h", want: time.Hour},
		{name: "one day", value: "1d", want: 24 * time.Hour},
		{name: "seven days", value: "7d", want: 7 * 24 * time.Hour},
		{name: "thirty days", value: "30d", want: 30 * 24 * time.Hour},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := paste.ParseAllowedTTL(tt.value)
			if err != nil {
				t.Fatalf("ParseAllowedTTL(%q) error = %v", tt.value, err)
			}
			if got != tt.want {
				t.Fatalf("ParseAllowedTTL(%q) = %v, want %v", tt.value, got, tt.want)
			}
		})
	}

	if _, err := paste.ParseAllowedTTL("2d"); !errors.Is(err, paste.ErrInvalidTTL) {
		t.Fatalf("ParseAllowedTTL(2d) error = %v, want ErrInvalidTTL", err)
	}
	if _, err := paste.ValidateTTL(31*24*time.Hour, paste.DefaultTTL, paste.MaxTTL); !errors.Is(err, paste.ErrInvalidTTL) {
		t.Fatalf("ValidateTTL(over max) error = %v, want ErrInvalidTTL", err)
	}
}
