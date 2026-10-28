package security

import (
	"errors"
	"strings"
	"testing"
)

func TestPasswordHasherHashesAndVerifiesPassword(t *testing.T) {
	hasher, err := NewPasswordHasher(DefaultPasswordParameters())
	if err != nil {
		t.Fatalf("new password hasher: %v", err)
	}

	encoded, err := hasher.Hash("senha-segura")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if !strings.HasPrefix(encoded, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected PHC string: %q", encoded)
	}

	valid, err := hasher.Verify("senha-segura", encoded)
	if err != nil || !valid {
		t.Fatalf("verify password = %v, %v; want true, nil", valid, err)
	}
	valid, err = hasher.Verify("senha-incorreta", encoded)
	if err != nil || valid {
		t.Fatalf("verify different password = %v, %v; want false, nil", valid, err)
	}
}

func TestValidatePasswordRejectsLengthAndWhitespace(t *testing.T) {
	for _, password := range []string{"curta", "senha com espaço", strings.Repeat("a", 33)} {
		if err := ValidatePassword(password); !errors.Is(err, ErrInvalidPassword) {
			t.Errorf("ValidatePassword(%q) = %v, want ErrInvalidPassword", password, err)
		}
	}
}

func TestPasswordHasherRejectsInvalidOrExcessivePHCParameters(t *testing.T) {
	hasher, err := NewPasswordHasher(DefaultPasswordParameters())
	if err != nil {
		t.Fatalf("new password hasher: %v", err)
	}
	for _, encoded := range []string{"invalid", "$argon2id$v=19$m=9999999,t=2,p=1$c2FsdHNhbHRzYWx0c2FsdA$YWJjZGVmZ2hpamtsbW5vcA"} {
		if _, err := hasher.Verify("senha-segura", encoded); !errors.Is(err, ErrInvalidPasswordHash) {
			t.Errorf("Verify invalid hash error = %v, want ErrInvalidPasswordHash", err)
		}
	}
}

func TestPasswordHasherDetectsParametersThatNeedRehash(t *testing.T) {
	current, err := NewPasswordHasher(DefaultPasswordParameters())
	if err != nil {
		t.Fatalf("new current hasher: %v", err)
	}
	olderParameters := DefaultPasswordParameters()
	olderParameters.MemoryKiB = 20 * 1024
	older, err := NewPasswordHasher(olderParameters)
	if err != nil {
		t.Fatalf("new older hasher: %v", err)
	}
	encoded, err := older.Hash("senha-segura")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	needsRehash, err := current.NeedsRehash(encoded)
	if err != nil || !needsRehash {
		t.Fatalf("NeedsRehash = %v, %v; want true, nil", needsRehash, err)
	}
}

func TestValidatePasswordUsesUnicodeWhiteSpace(t *testing.T) {
	for _, space := range []rune{'\u0085', '\u00a0', '\u1680', '\u2007', '\u2028', '\u202f', '\u3000'} {
		if err := ValidatePassword("abcd" + string(space) + "efgh"); !errors.Is(err, ErrInvalidPassword) {
			t.Errorf("accepted Unicode whitespace U+%04X", space)
		}
	}
	for _, character := range []rune{'\uFEFF', '\u200B'} {
		if err := ValidatePassword("abcd" + string(character) + "efgh"); err != nil {
			t.Errorf("rejected non-whitespace U+%04X", character)
		}
	}
}
