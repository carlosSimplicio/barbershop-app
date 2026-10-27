package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"

	"barbershop-app/backend/internal/establishment"

	"github.com/google/uuid"
)

type LocalStore struct {
	root *os.Root
}

var _ establishment.ObjectStorage = (*LocalStore)(nil)

func NewLocalStore(rootPath string) (*LocalStore, error) {
	if err := os.MkdirAll(rootPath, 0o700); err != nil {
		return nil, storageFailure(err)
	}
	root, err := os.OpenRoot(rootPath)
	if err != nil {
		return nil, storageFailure(err)
	}
	store := &LocalStore{root: root}
	if err := store.verifyWritableRoot(); err != nil {
		_ = root.Close()
		return nil, err
	}
	return store, nil
}

func (store *LocalStore) Close() error {
	return store.root.Close()
}

func (store *LocalStore) verifyWritableRoot() error {
	probeName := ".access-" + uuid.NewString()
	probeFile, err := store.root.OpenFile(probeName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return storageFailure(err)
	}
	if err := probeFile.Close(); err != nil {
		_ = store.root.Remove(probeName)
		return storageFailure(err)
	}
	if err := store.root.Remove(probeName); err != nil {
		return storageFailure(err)
	}
	return nil
}

func (store *LocalStore) Put(ctx context.Context, key establishment.ObjectKey, content io.Reader) (returnErr error) {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	if content == nil {
		return fmt.Errorf("%w: nil content reader", establishment.ErrStorageFailure)
	}
	if err := ctx.Err(); err != nil {
		return storageFailure(err)
	}

	temporaryName := ".write-" + uuid.NewString()
	temporaryFile, err := store.root.OpenFile(temporaryName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return storageFailure(err)
	}
	temporaryExists := true
	defer func() {
		if temporaryExists {
			if err := store.root.Remove(temporaryName); err != nil && !errors.Is(err, os.ErrNotExist) {
				returnErr = errors.Join(returnErr, storageFailure(err))
			}
		}
	}()

	if _, err := io.Copy(temporaryFile, contextReader{ctx: ctx, reader: content}); err != nil {
		_ = temporaryFile.Close()
		return storageFailure(err)
	}
	if err := ctx.Err(); err != nil {
		_ = temporaryFile.Close()
		return storageFailure(err)
	}
	if err := temporaryFile.Sync(); err != nil {
		_ = temporaryFile.Close()
		return storageFailure(err)
	}
	if err := temporaryFile.Close(); err != nil {
		return storageFailure(err)
	}

	if err := store.root.Link(temporaryName, string(key)); err != nil {
		if errors.Is(err, os.ErrExist) {
			return establishment.ErrObjectAlreadyExists
		}
		return storageFailure(err)
	}
	if err := store.root.Remove(temporaryName); err != nil {
		_ = store.root.Remove(string(key))
		return storageFailure(err)
	}
	temporaryExists = false
	return nil
}

func (store *LocalStore) Open(ctx context.Context, key establishment.ObjectKey) (io.ReadCloser, error) {
	if err := validateObjectKey(key); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, storageFailure(err)
	}
	file, err := store.root.Open(string(key))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, establishment.ErrObjectNotFound
		}
		return nil, storageFailure(err)
	}
	return file, nil
}

func (store *LocalStore) Delete(ctx context.Context, key establishment.ObjectKey) error {
	if err := validateObjectKey(key); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return storageFailure(err)
	}
	if err := store.root.Remove(string(key)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return storageFailure(err)
	}
	return nil
}

func validateObjectKey(key establishment.ObjectKey) error {
	parsed, err := uuid.Parse(string(key))
	if err != nil || parsed.String() != string(key) || parsed.Version() != uuid.Version(4) {
		return establishment.ErrInvalidObjectKey
	}
	return nil
}

func storageFailure(err error) error {
	return storageOperationError{cause: err}
}

type storageOperationError struct {
	cause error
}

func (storageOperationError) Error() string {
	return establishment.ErrStorageFailure.Error()
}

func (failure storageOperationError) Unwrap() error {
	return failure.cause
}

func (storageOperationError) Is(target error) bool {
	return target == establishment.ErrStorageFailure
}

type contextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (reader contextReader) Read(buffer []byte) (int, error) {
	if err := reader.ctx.Err(); err != nil {
		return 0, err
	}
	return reader.reader.Read(buffer)
}
