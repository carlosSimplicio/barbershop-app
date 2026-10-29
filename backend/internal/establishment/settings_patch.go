package establishment

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

var (
	ErrEmptySettingsPatch = errors.New("establishment settings patch is empty")
	ErrTimeZoneLocked     = errors.New("establishment time zone is locked")
)

type PatchField[T any] struct {
	Set   bool
	Null  bool
	Value T
}

type SettingsPatch struct {
	Name                          PatchField[string]
	TimeZone                      PatchField[string]
	Description                   PatchField[string]
	Address                       PatchField[string]
	ContactPhone                  PatchField[string]
	SocialLinks                   PatchField[[]SocialLink]
	MinimumBookingNoticeMinutes   PatchField[int32]
	BookingHorizonDays            PatchField[int32]
	CustomerChangeDeadlineMinutes PatchField[int32]
}

type SettingsFieldError struct {
	Field   string
	Code    string
	Message string
}

func (fieldError *SettingsFieldError) Error() string {
	return fmt.Sprintf("invalid establishment settings field %s: %s", fieldError.Field, fieldError.Code)
}

func validateSettingsPatch(patch SettingsPatch) (SettingsPatch, error) {
	if !patch.hasFields() {
		return SettingsPatch{}, ErrEmptySettingsPatch
	}
	if patch.Name.Set {
		if patch.Name.Null {
			return SettingsPatch{}, invalidSettingsField("name", "invalid_name", fmt.Sprintf("Informe um nome com até %d caracteres.", MaxNameCodePoints))
		}
		name := strings.TrimSpace(patch.Name.Value)
		if !IsNameValid(name) {
			return SettingsPatch{}, invalidSettingsField("name", "invalid_name", fmt.Sprintf("Informe um nome com até %d caracteres.", MaxNameCodePoints))
		}
		patch.Name.Value = name
	}
	if patch.TimeZone.Set {
		if patch.TimeZone.Null {
			return SettingsPatch{}, invalidSettingsField("time_zone", "invalid_time_zone", "Informe um fuso horário IANA válido.")
		}
		name := strings.TrimSpace(patch.TimeZone.Value)
		if !IsTimeZoneValid(name) {
			return SettingsPatch{}, invalidSettingsField("time_zone", "invalid_time_zone", "Informe um fuso horário IANA válido.")
		}
		patch.TimeZone.Value = name
	}
	for _, textField := range []struct {
		name  string
		field *PatchField[string]
	}{
		{name: "description", field: &patch.Description},
		{name: "address", field: &patch.Address},
	} {
		if !textField.field.Set || textField.field.Null {
			continue
		}
		value := strings.TrimSpace(textField.field.Value)
		if !utf8.ValidString(value) || utf8.RuneCountInString(value) > MaxProfileTextCodePoints {
			return SettingsPatch{}, invalidSettingsField(textField.name, "invalid_text", fmt.Sprintf("O texto deve ter até %d caracteres.", MaxProfileTextCodePoints))
		}
		if value == "" {
			textField.field.Null = true
			continue
		}
		textField.field.Value = value
	}
	if patch.ContactPhone.Set && !patch.ContactPhone.Null {
		phone, ok := normalizeBrazilianPhone(patch.ContactPhone.Value)
		if !ok {
			return SettingsPatch{}, invalidSettingsField("contact_phone", "invalid_phone", "Informe um telefone brasileiro com DDD.")
		}
		patch.ContactPhone.Value = phone
	}
	if patch.SocialLinks.Set {
		if patch.SocialLinks.Null {
			patch.SocialLinks.Value = []SocialLink{}
			patch.SocialLinks.Null = false
		}
		if len(patch.SocialLinks.Value) > MaxSocialLinks {
			return SettingsPatch{}, invalidSettingsField("social_links", "too_many_links", fmt.Sprintf("Informe no máximo %d links.", MaxSocialLinks))
		}
		for index, link := range patch.SocialLinks.Value {
			fieldPrefix := "social_links." + strconv.Itoa(index)
			label := strings.TrimSpace(link.Label)
			if !utf8.ValidString(label) || label == "" || utf8.RuneCountInString(label) > MaxSocialLinkLabelCodePoints {
				return SettingsPatch{}, invalidSettingsField(fieldPrefix+".label", "invalid_label", fmt.Sprintf("Informe um rótulo com até %d caracteres.", MaxSocialLinkLabelCodePoints))
			}
			linkURL := strings.TrimSpace(link.URL)
			if !validSocialLinkURL(linkURL) {
				return SettingsPatch{}, invalidSettingsField(fieldPrefix+".url", "invalid_url", "Informe uma URL HTTPS válida sem credenciais.")
			}
			patch.SocialLinks.Value[index] = SocialLink{Label: label, URL: linkURL}
		}
	}
	for _, limit := range []struct {
		name     string
		field    PatchField[int32]
		positive bool
	}{
		{name: "minimum_booking_notice_minutes", field: patch.MinimumBookingNoticeMinutes},
		{name: "customer_change_deadline_minutes", field: patch.CustomerChangeDeadlineMinutes},
		{name: "booking_horizon_days", field: patch.BookingHorizonDays, positive: true},
	} {
		if limit.field.Set && !limit.field.Null && (limit.field.Value < 0 || (limit.positive && limit.field.Value == 0)) {
			message := "Informe um número inteiro não negativo."
			code := "invalid_limit"
			if limit.positive {
				message = "Informe um número inteiro maior que zero."
				code = "invalid_horizon"
			}
			return SettingsPatch{}, invalidSettingsField(limit.name, code, message)
		}
	}
	return patch, nil
}

func (patch SettingsPatch) hasFields() bool {
	return patch.Name.Set || patch.TimeZone.Set || patch.Description.Set || patch.Address.Set || patch.ContactPhone.Set ||
		patch.SocialLinks.Set || patch.MinimumBookingNoticeMinutes.Set || patch.BookingHorizonDays.Set || patch.CustomerChangeDeadlineMinutes.Set
}

func invalidSettingsField(field, code, message string) *SettingsFieldError {
	return &SettingsFieldError{Field: field, Code: code, Message: message}
}

func normalizeBrazilianPhone(value string) (string, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	hasInternationalPrefix := strings.HasPrefix(value, "+")
	digits := make([]byte, 0, len(value))
	for index, character := range value {
		switch {
		case character >= '0' && character <= '9':
			digits = append(digits, byte(character))
		case character == '+' && index == 0:
		case character == ' ' || character == '\t' || character == '\n' || character == '\r' || character == '(' || character == ')' || character == '-' || character == '.':
		default:
			return "", false
		}
	}
	if hasInternationalPrefix {
		if len(digits) < 2 || string(digits[:2]) != "55" {
			return "", false
		}
		digits = digits[2:]
	} else if len(digits) == 12 || len(digits) == 13 {
		if string(digits[:2]) != "55" {
			return "", false
		}
		digits = digits[2:]
	}
	if len(digits) != 10 && len(digits) != 11 {
		return "", false
	}
	return "+55" + string(digits), true
}

func validSocialLinkURL(value string) bool {
	if value == "" || utf8.RuneCountInString(value) > MaxSocialLinkURLCodePoints {
		return false
	}
	parsed, err := url.Parse(value)
	return err == nil && strings.EqualFold(parsed.Scheme, "https") && parsed.Hostname() != "" && parsed.User == nil && parsed.Opaque == ""
}
