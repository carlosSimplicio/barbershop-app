package auth

import (
	"strings"
	"testing"
)

func TestValidateEmailAcceptsMailboxAndRejectsInvalidOrOversizedValues(t *testing.T) {
	for _, email := range []string{"owner@example.com", " Owner@Example.COM "} {
		if err := ValidateEmail(email); err != nil {
			t.Errorf("ValidateEmail(%q) = %v, want nil", email, err)
		}
	}
	for _, email := range []string{"", "owner", "Owner <owner@example.com>", strings.Repeat("a", 244) + "@example.com"} {
		if err := ValidateEmail(email); err == nil {
			t.Errorf("ValidateEmail(%q) = nil, want error", email)
		}
	}
}
