package establishment

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

var ErrPublicLogoNotFound = errors.New("public establishment logo not found")

type PublicLogo struct {
	ContentType string
	ByteSize    int64
	ETag        string
}

type publicLogoRecord struct {
	ObjectKey   ObjectKey
	ContentType string
	ByteSize    int64
	SHA256      []byte
}

type publicLogoStore interface {
	FindPublicLogo(context.Context, string, string) (publicLogoRecord, error)
}

type PublicLogoReader interface {
	OpenPublicLogo(context.Context, string) (PublicLogo, io.ReadCloser, error)
}

type PublicLogoService struct {
	resolver *PublicEstablishmentResolver
	store    publicLogoStore
	storage  ObjectStorage
}

func NewPublicLogoService(resolver *PublicEstablishmentResolver, store publicLogoStore, storage ObjectStorage) *PublicLogoService {
	return &PublicLogoService{resolver: resolver, store: store, storage: storage}
}

func (service *PublicLogoService) OpenPublicLogo(ctx context.Context, slug string) (PublicLogo, io.ReadCloser, error) {
	establishment, err := service.resolver.Resolve(ctx, slug)
	if errors.Is(err, ErrPublicEstablishmentNotFound) {
		return PublicLogo{}, nil, ErrPublicLogoNotFound
	}
	if err != nil {
		return PublicLogo{}, nil, err
	}
	record, err := service.store.FindPublicLogo(ctx, establishment.ID, slug)
	if err != nil {
		return PublicLogo{}, nil, fmt.Errorf("resolve public establishment logo: %w", err)
	}
	for attempt := 0; attempt < 2; attempt++ {
		logo, err := publicLogoFromRecord(record)
		if err != nil {
			return PublicLogo{}, nil, err
		}
		content, err := service.storage.Open(ctx, record.ObjectKey)
		if err == nil {
			return logo, content, nil
		}
		if !errors.Is(err, ErrObjectNotFound) || attempt == 1 {
			return PublicLogo{}, nil, fmt.Errorf("open public establishment logo: %w", err)
		}

		// A replacement can remove the old object after this request reads its key.
		record, err = service.store.FindPublicLogo(ctx, establishment.ID, slug)
		if errors.Is(err, ErrPublicLogoNotFound) {
			return PublicLogo{}, nil, ErrPublicLogoNotFound
		}
		if err != nil {
			return PublicLogo{}, nil, fmt.Errorf("re-resolve public establishment logo: %w", err)
		}
	}
	return PublicLogo{}, nil, errors.New("public establishment logo could not be opened")
}

func publicLogoFromRecord(record publicLogoRecord) (PublicLogo, error) {
	if record.ByteSize <= 0 || record.ByteSize > MaxLogoBytes || len(record.SHA256) != sha256.Size || !isSupportedLogoMediaType(record.ContentType) {
		return PublicLogo{}, errors.New("public establishment logo metadata is invalid")
	}
	return PublicLogo{
		ContentType: record.ContentType,
		ByteSize:    record.ByteSize,
		ETag:        `"` + hex.EncodeToString(record.SHA256) + `"`,
	}, nil
}
