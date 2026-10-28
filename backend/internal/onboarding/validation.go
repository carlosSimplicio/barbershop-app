package onboarding

import (
	"errors"

	"barbershop-app/backend/internal/establishment"
)

var (
	ErrInvalidSlug     = errors.New("invalid establishment slug")
	ErrReservedSlug    = errors.New("reserved establishment slug")
	ErrInvalidTimeZone = errors.New("invalid establishment time zone")
	ErrInvalidName     = errors.New("invalid establishment name")
)

func ValidateEstablishmentName(name string) error {
	if !establishment.IsNameValid(name) {
		return ErrInvalidName
	}
	return nil
}

// ValidateSlug validates the immutable public identifier chosen during onboarding.
func ValidateSlug(slug string) error {
	if !establishment.IsPublicSlugFormatValid(slug) {
		return ErrInvalidSlug
	}
	if establishment.IsPublicSlugReserved(slug) {
		return ErrReservedSlug
	}
	return nil
}

// ValidateTimeZone validates the establishment IANA time zone.
func ValidateTimeZone(name string) error {
	if !establishment.IsTimeZoneValid(name) {
		return ErrInvalidTimeZone
	}
	return nil
}
