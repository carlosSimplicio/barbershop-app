package establishment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"strings"
	"testing"
)

func TestPublicLogoServiceRetriesWithCurrentKeyAfterConcurrentReplacement(t *testing.T) {
	oldHash := sha256.Sum256([]byte("old logo"))
	newHash := sha256.Sum256([]byte("new logo"))
	store := &rotatingPublicLogoStore{records: []publicLogoRecord{
		{ObjectKey: "old-key", ContentType: "image/png", ByteSize: 8, SHA256: oldHash[:]},
		{ObjectKey: "new-key", ContentType: "image/png", ByteSize: 8, SHA256: newHash[:]},
	}}
	storage := &replacingPublicLogoStorage{}
	service := NewPublicLogoService(
		NewPublicEstablishmentResolver(publicEstablishmentStoreStub{}), store, storage,
	)

	logo, content, err := service.OpenPublicLogo(context.Background(), "barbearia-central")
	if err != nil {
		t.Fatalf("OpenPublicLogo() error = %v", err)
	}
	defer content.Close()
	body, err := io.ReadAll(content)
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != "new logo" || logo.ETag != `"`+hex.EncodeToString(newHash[:])+`"` {
		t.Fatalf("logo body/ETag = %q/%q; want current bytes and hash", body, logo.ETag)
	}
	if len(storage.opened) != 2 || storage.opened[0] != "old-key" || storage.opened[1] != "new-key" {
		t.Fatalf("opened keys = %v; want old then current key", storage.opened)
	}
}

type publicEstablishmentStoreStub struct{}

func (publicEstablishmentStoreStub) FindPublicEstablishmentBySlug(_ context.Context, slug string) (PublicEstablishment, error) {
	return PublicEstablishment{ID: testEstablishmentID, Name: "Barbearia Central", Slug: slug}, nil
}

type rotatingPublicLogoStore struct {
	records []publicLogoRecord
	reads   int
}

func (store *rotatingPublicLogoStore) FindPublicLogo(context.Context, string, string) (publicLogoRecord, error) {
	index := store.reads
	store.reads++
	if index >= len(store.records) {
		index = len(store.records) - 1
	}
	return store.records[index], nil
}

type replacingPublicLogoStorage struct {
	opened []ObjectKey
}

func (storage *replacingPublicLogoStorage) Put(context.Context, ObjectKey, io.Reader) error {
	return nil
}

func (storage *replacingPublicLogoStorage) Open(_ context.Context, key ObjectKey) (io.ReadCloser, error) {
	storage.opened = append(storage.opened, key)
	if key == "old-key" {
		return nil, ErrObjectNotFound
	}
	return io.NopCloser(strings.NewReader("new logo")), nil
}

func (storage *replacingPublicLogoStorage) Delete(context.Context, ObjectKey) error { return nil }
