package establishment

import (
	"context"
	"errors"
	"fmt"
	"time"

	"barbershop-app/backend/internal/auth"
	"barbershop-app/backend/internal/entitlements"
)

var ErrSettingsForbidden = errors.New("establishment settings access forbidden")

type TrialStatus string

const (
	TrialStatusActive  TrialStatus = "active"
	TrialStatusExpired TrialStatus = "expired"
)

type Trial struct {
	Status   TrialStatus
	StartsAt time.Time
	EndsAt   time.Time
}

type Establishment struct {
	ID       string
	Name     string
	Slug     string
	TimeZone string
	Trial    *Trial
}

type SocialLink struct {
	Label string
	URL   string
}

type LogoMetadata struct {
	URL         *string
	ContentType string
	ByteSize    int64
	Width       int32
	Height      int32
}

type EstablishmentSettings struct {
	Establishment
	Description                   *string
	Address                       *string
	ContactPhone                  *string
	SocialLinks                   []SocialLink
	MinimumBookingNoticeMinutes   *int32
	BookingHorizonDays            *int32
	CustomerChangeDeadlineMinutes *int32
	OperationalDataStarted        bool
	Logo                          *LogoMetadata
	TimeZoneEditable              bool
	CanWrite                      bool
	SlugConflict                  bool
	IsComplete                    bool
	IsPublishable                 bool
}

type Store interface {
	GetByID(context.Context, string) (Establishment, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

type SettingsStore interface {
	GetSettingsByID(context.Context, string) (EstablishmentSettings, error)
	UpdateSettings(context.Context, string, SettingsPatch) error
}

type WriteEntitlements interface {
	RequireWrite(context.Context, string) error
}

type SettingsService struct {
	store        SettingsStore
	entitlements WriteEntitlements
}

func NewSettingsService(store SettingsStore, writeEntitlements WriteEntitlements) *SettingsService {
	return &SettingsService{store: store, entitlements: writeEntitlements}
}

func (service *SettingsService) GetSettings(ctx context.Context, principal auth.Principal) (EstablishmentSettings, error) {
	if principal.Role != "owner" {
		return EstablishmentSettings{}, ErrSettingsForbidden
	}
	settings, err := service.store.GetSettingsByID(ctx, principal.EstablishmentID)
	if err != nil {
		return EstablishmentSettings{}, fmt.Errorf("get authenticated establishment settings: %w", err)
	}
	settings.SlugConflict = IsPublicSlugReserved(settings.Slug)
	settings.IsComplete = settings.Logo != nil
	settings.IsPublishable = settings.IsComplete && !settings.SlugConflict
	if err := service.entitlements.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		if !errors.Is(err, entitlements.ErrReadOnly) {
			return EstablishmentSettings{}, fmt.Errorf("check establishment write access: %w", err)
		}
	} else {
		settings.CanWrite = true
	}
	settings.TimeZoneEditable = !settings.OperationalDataStarted && settings.CanWrite
	if settings.Logo != nil && settings.IsPublishable {
		logoURL := "/api/v1/public/establishments/" + settings.Slug + "/logo"
		settings.Logo.URL = &logoURL
	}
	return settings, nil
}

func (service *SettingsService) PatchSettings(ctx context.Context, principal auth.Principal, patch SettingsPatch) error {
	if principal.Role != "owner" {
		return ErrSettingsForbidden
	}
	validatedPatch, err := validateSettingsPatch(patch)
	if err != nil {
		return err
	}
	if err := service.entitlements.RequireWrite(ctx, principal.EstablishmentID); err != nil {
		return err
	}
	if err := service.store.UpdateSettings(ctx, principal.EstablishmentID, validatedPatch); err != nil {
		return fmt.Errorf("update authenticated establishment settings: %w", err)
	}
	return nil
}

func (service *Service) Get(ctx context.Context, principal auth.Principal) (Establishment, error) {
	establishment, err := service.store.GetByID(ctx, principal.EstablishmentID)
	if err != nil {
		return Establishment{}, fmt.Errorf("get authenticated establishment: %w", err)
	}
	if establishment.Trial == nil {
		return establishment, nil
	}

	establishment.Trial.StartsAt = establishment.Trial.StartsAt.UTC()
	establishment.Trial.EndsAt = establishment.Trial.EndsAt.UTC()
	now := service.now().UTC()
	if !now.Before(establishment.Trial.StartsAt) && now.Before(establishment.Trial.EndsAt) {
		establishment.Trial.Status = TrialStatusActive
	} else {
		establishment.Trial.Status = TrialStatusExpired
	}
	return establishment, nil
}
