package catalog

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	ErrInvalidName     = errors.New("invalid offering name")
	ErrInvalidPrice    = errors.New("invalid offering price")
	ErrInvalidDuration = errors.New("invalid offering duration")
)

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if !utf8.ValidString(value) || value == "" || strings.ContainsRune(value, 0) || utf8.RuneCountInString(value) > MaxNameCodePoints {
		return "", ErrInvalidName
	}
	return value, nil
}
func validatePrice(value int64) error {
	if value < MinPriceCents || value > MaxPriceCents {
		return ErrInvalidPrice
	}
	return nil
}
func normalizeDuration(value *int64) (int32, error) {
	if value == nil {
		return DefaultDurationMinutes, nil
	}
	if *value < MinDurationMinutes || *value > MaxDurationMinutes {
		return 0, ErrInvalidDuration
	}
	return int32(*value), nil
}
