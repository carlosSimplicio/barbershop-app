package onboarding

import (
	"errors"
	"regexp"
	"time"
	_ "time/tzdata"
)

var (
	ErrInvalidSlug     = errors.New("invalid establishment slug")
	ErrReservedSlug    = errors.New("reserved establishment slug")
	ErrInvalidTimeZone = errors.New("invalid establishment time zone")

	validSlug     = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{1,61}[a-z0-9])$`)
	reservedSlugs = map[string]struct{}{
		"admin": {}, "api": {}, "login": {}, "assets": {},
	}
)

// ValidateSlug validates the immutable public identifier chosen during onboarding.
func ValidateSlug(slug string) error {
	if !validSlug.MatchString(slug) {
		return ErrInvalidSlug
	}
	if _, reserved := reservedSlugs[slug]; reserved {
		return ErrReservedSlug
	}
	return nil
}

// ValidateTimeZone validates the establishment IANA time zone.
func ValidateTimeZone(name string) error {
	if name == "" || name == "Local" {
		return ErrInvalidTimeZone
	}
	if _, err := time.LoadLocation(name); err != nil {
		return ErrInvalidTimeZone
	}
	return nil
}
