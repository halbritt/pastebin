package publishauth

import (
	"strings"
	"testing"
)

func TestValidateToken(t *testing.T) {
	for _, tt := range []struct {
		name  string
		token string
		valid bool
	}{
		{name: "three words", token: "almanac-breeze-copper", valid: true},
		{name: "short random token", token: "AQIDBAUGBwgJCgsMDQ4PEA", valid: true},
		{name: "existing long token", token: strings.Repeat("a", 64), valid: true},
		{name: "too short", token: "too-short", valid: false},
		{name: "malformed short token", token: strings.Repeat("!", 22), valid: false},
		{name: "malformed three words", token: "almanac-Breeze-copper", valid: false},
		{name: "unsupported intermediate length", token: strings.Repeat("a", 23), valid: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateToken(tt.token)
			if (err == nil) != tt.valid {
				t.Fatalf("ValidateToken() error = %v, valid = %t", err, tt.valid)
			}
		})
	}
}
