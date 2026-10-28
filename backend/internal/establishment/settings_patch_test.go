package establishment

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateSettingsPatchNormalizesTextPhoneAndSocialLinks(t *testing.T) {
	patch, err := validateSettingsPatch(SettingsPatch{
		Name:         PatchField[string]{Set: true, Value: "  Barbearia Central  "},
		TimeZone:     PatchField[string]{Set: true, Value: " America/Recife "},
		Description:  PatchField[string]{Set: true, Value: "  Cortes e barba  "},
		Address:      PatchField[string]{Set: true, Value: "   "},
		ContactPhone: PatchField[string]{Set: true, Value: "+55 (11) 99999-8888"},
		SocialLinks: PatchField[[]SocialLink]{Set: true, Value: []SocialLink{{
			Label: " Instagram ", URL: " https://instagram.com/barbearia ",
		}}},
		MinimumBookingNoticeMinutes: PatchField[int32]{Set: true, Value: 0},
		BookingHorizonDays:          PatchField[int32]{Set: true, Value: 30},
	})

	if err != nil {
		t.Fatal(err)
	}
	if patch.Name.Value != "Barbearia Central" || patch.TimeZone.Value != "America/Recife" ||
		patch.Description.Value != "Cortes e barba" || !patch.Address.Null || patch.ContactPhone.Value != "+5511999998888" ||
		len(patch.SocialLinks.Value) != 1 || patch.SocialLinks.Value[0].Label != "Instagram" || patch.SocialLinks.Value[0].URL != "https://instagram.com/barbearia" {
		t.Fatalf("normalized patch = %#v", patch)
	}
}

func TestValidateSettingsPatchTreatsNullLinksAsClear(t *testing.T) {
	patch, err := validateSettingsPatch(SettingsPatch{SocialLinks: PatchField[[]SocialLink]{Set: true, Null: true}})

	if err != nil {
		t.Fatal(err)
	}
	if !patch.SocialLinks.Set || patch.SocialLinks.Null || len(patch.SocialLinks.Value) != 0 {
		t.Fatalf("social links patch = %#v", patch.SocialLinks)
	}
}

func TestValidateSettingsPatchRejectsInvalidValuesByField(t *testing.T) {
	for _, test := range []struct {
		name  string
		patch SettingsPatch
		field string
	}{
		{name: "empty patch", patch: SettingsPatch{}, field: ""},
		{name: "blank name", patch: SettingsPatch{Name: PatchField[string]{Set: true, Value: "  "}}, field: "name"},
		{name: "long name", patch: SettingsPatch{Name: PatchField[string]{Set: true, Value: strings.Repeat("a", 121)}}, field: "name"},
		{name: "long description", patch: SettingsPatch{Description: PatchField[string]{Set: true, Value: strings.Repeat("á", 501)}}, field: "description"},
		{name: "invalid timezone", patch: SettingsPatch{TimeZone: PatchField[string]{Set: true, Value: "Local"}}, field: "time_zone"},
		{name: "invalid phone", patch: SettingsPatch{ContactPhone: PatchField[string]{Set: true, Value: "not a phone"}}, field: "contact_phone"},
		{name: "too many links", patch: SettingsPatch{SocialLinks: PatchField[[]SocialLink]{Set: true, Value: make([]SocialLink, 6)}}, field: "social_links"},
		{name: "blank label", patch: SettingsPatch{SocialLinks: PatchField[[]SocialLink]{Set: true, Value: []SocialLink{{Label: " ", URL: "https://example.test"}}}}, field: "social_links.0.label"},
		{name: "non https link", patch: SettingsPatch{SocialLinks: PatchField[[]SocialLink]{Set: true, Value: []SocialLink{{Label: "Site", URL: "http://example.test"}}}}, field: "social_links.0.url"},
		{name: "link credentials", patch: SettingsPatch{SocialLinks: PatchField[[]SocialLink]{Set: true, Value: []SocialLink{{Label: "Site", URL: "https://user:secret@example.test"}}}}, field: "social_links.0.url"},
		{name: "long url", patch: SettingsPatch{SocialLinks: PatchField[[]SocialLink]{Set: true, Value: []SocialLink{{Label: "Site", URL: "https://example.test/" + strings.Repeat("a", 2049)}}}}, field: "social_links.0.url"},
		{name: "negative notice", patch: SettingsPatch{MinimumBookingNoticeMinutes: PatchField[int32]{Set: true, Value: -1}}, field: "minimum_booking_notice_minutes"},
		{name: "zero horizon", patch: SettingsPatch{BookingHorizonDays: PatchField[int32]{Set: true, Value: 0}}, field: "booking_horizon_days"},
		{name: "negative deadline", patch: SettingsPatch{CustomerChangeDeadlineMinutes: PatchField[int32]{Set: true, Value: -1}}, field: "customer_change_deadline_minutes"},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := validateSettingsPatch(test.patch)
			if test.field == "" {
				if !errors.Is(err, ErrEmptySettingsPatch) {
					t.Fatalf("error = %v, want ErrEmptySettingsPatch", err)
				}
				return
			}
			var fieldError *SettingsFieldError
			if !errors.As(err, &fieldError) || fieldError.Field != test.field {
				t.Fatalf("error = %v, want field %q", err, test.field)
			}
		})
	}
}

func TestNormalizeBrazilianPhoneAddsCountryCodeAndPreservesProvidedCode(t *testing.T) {
	for _, test := range []struct {
		input string
		want  string
	}{
		{input: "(11) 99999-8888", want: "+5511999998888"},
		{input: "+55 11 3333-2222", want: "+551133332222"},
		{input: "5511999998888", want: "+5511999998888"},
	} {
		got, ok := normalizeBrazilianPhone(test.input)
		if !ok || got != test.want {
			t.Errorf("normalizeBrazilianPhone(%q) = %q, %v; want %q, true", test.input, got, ok, test.want)
		}
	}
}

func TestNormalizeBrazilianPhoneRejectsOtherExplicitInternationalCodes(t *testing.T) {
	for _, input := range []string{"+1 1234567890", "+44 20 1234 5678"} {
		if normalized, ok := normalizeBrazilianPhone(input); ok {
			t.Errorf("normalizeBrazilianPhone(%q) = %q, true; want rejection", input, normalized)
		}
	}
}
