package establishment

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"barbershop-app/backend/internal/auth"
)

var (
	ErrLogoUploadForbidden      = errors.New("establishment logo upload forbidden")
	ErrLogoAssociationUncertain = errors.New("establishment logo association uncertain")
)

type LogoUploadStore interface {
	ReplaceCurrentLogo(context.Context, string, ObjectKey, ValidatedLogo) (ObjectKey, error)
	CurrentLogoObjectKey(context.Context, string) (ObjectKey, error)
}

type LogoUploader interface {
	CheckLogoUploadAccess(context.Context, auth.Principal) error
	UploadLogo(context.Context, auth.Principal, ValidatedLogo) error
}

type LogoUploadService struct {
	store        LogoUploadStore
	storage      ObjectStorage
	entitlements WriteEntitlements
}

func NewLogoUploadService(store LogoUploadStore, storage ObjectStorage, entitlements WriteEntitlements) *LogoUploadService {
	return &LogoUploadService{store: store, storage: storage, entitlements: entitlements}
}

func (service *LogoUploadService) CheckLogoUploadAccess(ctx context.Context, principal auth.Principal) error {
	if principal.Role != "owner" {
		return ErrLogoUploadForbidden
	}
	return service.entitlements.RequireWrite(ctx, principal.EstablishmentID)
}

func (service *LogoUploadService) UploadLogo(ctx context.Context, principal auth.Principal, logo ValidatedLogo) error {
	if err := service.CheckLogoUploadAccess(ctx, principal); err != nil {
		return err
	}
	key, err := NewObjectKey()
	if err != nil {
		return err
	}
	if err := service.storage.Put(ctx, key, logo.ContentReader()); err != nil {
		return fmt.Errorf("store establishment logo: %w", err)
	}

	previousKey, err := service.store.ReplaceCurrentLogo(ctx, principal.EstablishmentID, key, logo)
	if err != nil {
		currentKey, lookupErr := service.store.CurrentLogoObjectKey(ctx, principal.EstablishmentID)
		if lookupErr != nil {
			slog.Error("establishment_logo_association_uncertain")
			return errors.Join(ErrLogoAssociationUncertain, err, lookupErr)
		}
		if currentKey != key {
			service.compensate(ctx, key, "establishment_logo_upload_compensation_failed")
			return fmt.Errorf("associate establishment logo: %w", err)
		}
		// The commit may have succeeded even though its acknowledgement was lost.
	}

	if previousKey != "" && previousKey != key {
		service.compensate(ctx, previousKey, "establishment_logo_cleanup_failed")
	}
	return nil
}

func (service *LogoUploadService) compensate(ctx context.Context, key ObjectKey, event string) {
	cleanupContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := service.storage.Delete(cleanupContext, key); err != nil {
		slog.Error(event)
	}
}
