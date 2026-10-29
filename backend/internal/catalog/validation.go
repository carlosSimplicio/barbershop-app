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
	if !utf8.ValidString(value) || value == "" || strings.ContainsRune(value, 0) || utf8.RuneCountInString(value) > 120 {
		return "", ErrInvalidName
	}
	return value, nil
}
func validatePrice(value int64) error {
	if value < 0 || value > 9999999 {
		return ErrInvalidPrice
	}
	return nil
}
func normalizeDuration(value *int64) (int32, error) {
	if value == nil {
		return 30, nil
	}
	if *value < 1 || *value > 1440 {
		return 0, ErrInvalidDuration
	}
	return int32(*value), nil
}
