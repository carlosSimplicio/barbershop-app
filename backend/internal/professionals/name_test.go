package professionals

import (
	"strings"
	"testing"
)

func TestProfessionalNameUsesUnicodeLimitAndRejectsUnsafeValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		value string
		valid bool
	}{
		{"unicode boundary", strings.Repeat("á", MaxNameCodePoints), true},
		{"unicode over limit", strings.Repeat("á", MaxNameCodePoints+1), false},
		{"blank", " \t ", false}, {"NUL", "Ana\x00", false}, {"invalid UTF8", string([]byte{0xff}), false}, {"trimmed", " Ana ", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if IsNameValid(test.value) != test.valid {
				t.Fatalf("valid = %v", !test.valid)
			}
		})
	}
}
