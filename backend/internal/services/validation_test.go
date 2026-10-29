package services

import (
	"errors"
	"strings"
	"testing"
)

func TestServiceNameUsesTrimmedUnicodeLength(t *testing.T) {
	for _, tc := range []struct {
		input, want string
		invalid     bool
	}{
		{"  Corte  ", "Corte", false}, {"   ", "", true},
		{" " + strings.Repeat("á", 120) + " ", strings.Repeat("á", 120), false},
		{strings.Repeat("á", 121), "", true}, {string([]byte{0xff}), "", true},
	} {
		got, err := normalizeName(tc.input)
		if tc.invalid {
			if !errors.Is(err, ErrInvalidName) {
				t.Errorf("expected invalid name, got %v", err)
			}
			continue
		}
		if err != nil || got != tc.want {
			t.Errorf("normalization mismatch: %v", err)
		}
	}
}
func TestServicePriceIsBoundedInWholeCents(t *testing.T) {
	for _, v := range []int64{0, 1, 9999999} {
		if err := validatePrice(v); err != nil {
			t.Errorf("valid price %d: %v", v, err)
		}
	}
	for _, v := range []int64{-1, 10000000, 1 << 40} {
		if !errors.Is(validatePrice(v), ErrInvalidPrice) {
			t.Errorf("accepted price %d", v)
		}
	}
}
func TestServiceDurationDefaultAndMinutePrecision(t *testing.T) {
	if got, err := normalizeDuration(nil); err != nil || got != 30 {
		t.Fatalf("default=%d err=%v", got, err)
	}
	for _, v := range []int64{1, 17, 1440} {
		got, err := normalizeDuration(&v)
		if err != nil || int64(got) != v {
			t.Errorf("duration %d: %v", v, err)
		}
	}
	for _, v := range []int64{-1, 0, 1441, 1 << 40} {
		if _, err := normalizeDuration(&v); !errors.Is(err, ErrInvalidDuration) {
			t.Errorf("accepted duration %d", v)
		}
	}
}
