package establishment

import (
	"context"
	"errors"
	"io"

	"github.com/google/uuid"
)

var (
	ErrInvalidObjectKey    = errors.New("invalid object key")
	ErrObjectNotFound      = errors.New("object not found")
	ErrObjectAlreadyExists = errors.New("object already exists")
	ErrStorageFailure      = errors.New("storage operation failed")
)

// ObjectKey is an opaque, application-generated reference to stored content.
type ObjectKey string

// NewObjectKey creates a random UUID v4 key for application-created objects.
func NewObjectKey() (ObjectKey, error) {
	id, err := uuid.NewRandom()
	if err != nil {
		return "", ErrStorageFailure
	}
	return ObjectKey(id.String()), nil
}

// ObjectStorage defines provider-neutral object operations for establishment use cases.
type ObjectStorage interface {
	Put(context.Context, ObjectKey, io.Reader) error
	Open(context.Context, ObjectKey) (io.ReadCloser, error)
	Delete(context.Context, ObjectKey) error
}
