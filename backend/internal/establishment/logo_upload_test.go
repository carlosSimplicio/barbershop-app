package establishment

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

	"barbershop-app/backend/internal/auth"
)

func TestLogoUploadServiceStoresBeforeAssociationAndRemovesReplacedObject(t *testing.T) {
	previous := ObjectKey("previous-key")
	store := &logoUploadStoreStub{previous: previous}
	objects := &logoObjectStorageStub{}
	service := NewLogoUploadService(store, objects, logoWriteEntitlementsStub{})

	if err := service.UploadLogo(context.Background(), auth.Principal{EstablishmentID: "tenant", Role: "owner"}, validatedLogoPNG(t)); err != nil {
		t.Fatal(err)
	}
	if len(objects.putKeys) != 1 || store.associatedKey != objects.putKeys[0] {
		t.Fatalf("object writes=%v, associated=%q; want object written before association", objects.putKeys, store.associatedKey)
	}
	if len(objects.deleted) != 1 || objects.deleted[0] != previous {
		t.Fatalf("deleted keys = %v, want previous object %q", objects.deleted, previous)
	}
}

func TestLogoUploadServicePreservesNewObjectWhenAssociationOutcomeIsUnknown(t *testing.T) {
	store := &logoUploadStoreStub{replaceErr: errors.New("commit acknowledgement lost"), lookupErr: errors.New("database unavailable")}
	objects := &logoObjectStorageStub{}
	service := NewLogoUploadService(store, objects, logoWriteEntitlementsStub{})

	err := service.UploadLogo(context.Background(), auth.Principal{EstablishmentID: "tenant", Role: "owner"}, validatedLogoPNG(t))
	if !errors.Is(err, ErrLogoAssociationUncertain) {
		t.Fatalf("error = %v, want uncertain association", err)
	}
	if len(objects.deleted) != 0 {
		t.Fatalf("deleted keys = %v, want no deletion while association is uncertain", objects.deleted)
	}
}

func TestLogoUploadServiceRemovesPreviousObjectAfterConfirmedCommitDespiteLostAcknowledgement(t *testing.T) {
	previous := ObjectKey("previous-key")
	store := &logoUploadStoreStub{previous: previous, confirmCommit: true, replaceErr: errors.New("commit acknowledgement lost")}
	objects := &logoObjectStorageStub{}
	service := NewLogoUploadService(store, objects, logoWriteEntitlementsStub{})

	if err := service.UploadLogo(context.Background(), auth.Principal{EstablishmentID: "tenant", Role: "owner"}, validatedLogoPNG(t)); err != nil {
		t.Fatal(err)
	}
	if len(objects.deleted) != 1 || objects.deleted[0] != previous {
		t.Fatalf("deleted keys = %v, want previous object removed after current reference confirms commit", objects.deleted)
	}
}

func TestLogoUploadServiceCompensatesUnassociatedObjectAndKeepsCurrentOnFailure(t *testing.T) {
	store := &logoUploadStoreStub{current: "previous-key", replaceErr: errors.New("database rejected association")}
	objects := &logoObjectStorageStub{}
	service := NewLogoUploadService(store, objects, logoWriteEntitlementsStub{})

	err := service.UploadLogo(context.Background(), auth.Principal{EstablishmentID: "tenant", Role: "owner"}, validatedLogoPNG(t))
	if err == nil {
		t.Fatal("expected association error")
	}
	if len(objects.deleted) != 1 || objects.deleted[0] != objects.putKeys[0] {
		t.Fatalf("compensated keys = %v, want newly uploaded key %q", objects.deleted, objects.putKeys[0])
	}
}

func TestLogoUploadServiceDoesNotAssociateWhenObjectStorageWriteFails(t *testing.T) {
	store := &logoUploadStoreStub{}
	objects := &logoObjectStorageStub{putErr: errors.New("storage write failed")}
	service := NewLogoUploadService(store, objects, logoWriteEntitlementsStub{})

	if err := service.UploadLogo(context.Background(), auth.Principal{EstablishmentID: "tenant", Role: "owner"}, validatedLogoPNG(t)); err == nil {
		t.Fatal("expected storage write error")
	}
	if store.replaceCalls != 0 {
		t.Fatalf("logo association calls = %d, want none before the object is complete", store.replaceCalls)
	}
}

func TestLogoUploadServiceRequiresOwnerWriteAccessAndValidImage(t *testing.T) {
	objects := &logoObjectStorageStub{}
	store := &logoUploadStoreStub{}
	service := NewLogoUploadService(store, objects, logoWriteEntitlementsStub{err: errors.New("write denied")})
	if err := service.UploadLogo(context.Background(), auth.Principal{Role: "owner"}, validatedLogoPNG(t)); err == nil {
		t.Fatal("expected write access error")
	}
	if len(objects.putKeys) != 0 {
		t.Fatal("stored object without write access")
	}

	service = NewLogoUploadService(store, objects, logoWriteEntitlementsStub{})
	if err := service.UploadLogo(context.Background(), auth.Principal{Role: "professional"}, validatedLogoPNG(t)); !errors.Is(err, ErrLogoUploadForbidden) {
		t.Fatalf("professional upload error = %v", err)
	}
	if len(objects.putKeys) != 0 {
		t.Fatal("stored a logo without write access or owner role")
	}
}

type logoUploadStoreStub struct {
	previous      ObjectKey
	current       ObjectKey
	associatedKey ObjectKey
	confirmCommit bool
	replaceErr    error
	lookupErr     error
	replaceCalls  int
}

func (store *logoUploadStoreStub) ReplaceCurrentLogo(_ context.Context, _ string, key ObjectKey, _ ValidatedLogo) (ObjectKey, error) {
	store.replaceCalls++
	store.associatedKey = key
	if store.confirmCommit {
		store.current = key
	}
	return store.previous, store.replaceErr
}

func (store *logoUploadStoreStub) CurrentLogoObjectKey(context.Context, string) (ObjectKey, error) {
	return store.current, store.lookupErr
}

type logoObjectStorageStub struct {
	putKeys []ObjectKey
	deleted []ObjectKey
	putErr  error
}

func (storage *logoObjectStorageStub) Put(_ context.Context, key ObjectKey, reader io.Reader) error {
	if storage.putErr != nil {
		return storage.putErr
	}
	if _, err := io.Copy(io.Discard, reader); err != nil {
		return err
	}
	storage.putKeys = append(storage.putKeys, key)
	return nil
}

func (*logoObjectStorageStub) Open(context.Context, ObjectKey) (io.ReadCloser, error) {
	return nil, ErrObjectNotFound
}

func (storage *logoObjectStorageStub) Delete(_ context.Context, key ObjectKey) error {
	storage.deleted = append(storage.deleted, key)
	return nil
}

type logoWriteEntitlementsStub struct{ err error }

func (entitlements logoWriteEntitlementsStub) RequireWrite(context.Context, string) error {
	return entitlements.err
}

func validatedLogoPNG(t *testing.T) ValidatedLogo {
	t.Helper()
	var content bytes.Buffer
	imageData := image.NewRGBA(image.Rect(0, 0, 1, 1))
	imageData.Set(0, 0, color.NRGBA{R: 32, G: 64, B: 96, A: 255})
	if err := png.Encode(&content, imageData); err != nil {
		t.Fatal(err)
	}
	validated, err := ValidateLogo(bytes.NewReader(content.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	return validated
}
