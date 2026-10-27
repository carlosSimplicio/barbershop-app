package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"barbershop-app/backend/internal/establishment"
)

func TestLocalStoreSatisfiesObjectStorageContract(t *testing.T) {
	store, _ := newTestLocalStore(t)
	assertObjectStorageContract(t, store)
}

func TestNewLocalStoreRejectsUnusableRootAndCleansAccessProbe(t *testing.T) {
	root := t.TempDir()
	store, err := NewLocalStore(root)
	if err != nil {
		t.Fatalf("create local store: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close local store: %v", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("list initialized storage directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("storage initialization left %d probe files", len(entries))
	}

	pathToFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(pathToFile, []byte("file"), 0o600); err != nil {
		t.Fatalf("create invalid storage root: %v", err)
	}
	if _, err := NewLocalStore(pathToFile); !errors.Is(err, establishment.ErrStorageFailure) {
		t.Fatalf("initialize file as storage root error = %v, want ErrStorageFailure", err)
	}
}

// assertObjectStorageContract is shared by storage adapters so provider implementations
// exercise the same semantics for writes, reads, missing objects, and removal.
func assertObjectStorageContract(t *testing.T, store establishment.ObjectStorage) {
	t.Helper()
	ctx := context.Background()
	key := newTestObjectKey(t)
	content := []byte("logo bytes")
	if err := store.Put(ctx, key, bytes.NewReader(content)); err != nil {
		t.Fatalf("put object: %v", err)
	}
	reader, err := store.Open(ctx, key)
	if err != nil {
		t.Fatalf("open stored object: %v", err)
	}
	actual, readErr := io.ReadAll(reader)
	closeErr := reader.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("read stored object: read=%v close=%v", readErr, closeErr)
	}
	if !bytes.Equal(actual, content) {
		t.Fatalf("stored content = %q, want %q", actual, content)
	}
	if err := store.Put(ctx, key, bytes.NewReader([]byte("replacement"))); !errors.Is(err, establishment.ErrObjectAlreadyExists) {
		t.Fatalf("put existing key error = %v, want ErrObjectAlreadyExists", err)
	}
	assertObjectContent(t, store, key, content)
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete object: %v", err)
	}
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("delete already absent object: %v", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, establishment.ErrObjectNotFound) {
		t.Fatalf("open deleted object error = %v, want ErrObjectNotFound", err)
	}
}

func TestLocalStoreFailedWriteRemovesTemporaryAndFinalObjects(t *testing.T) {
	store, root := newTestLocalStore(t)
	key := newTestObjectKey(t)
	writeErr := errors.New("source failed")
	err := store.Put(context.Background(), key, &failingReader{err: writeErr})
	if !errors.Is(err, establishment.ErrStorageFailure) {
		t.Fatalf("failed put error = %v, want ErrStorageFailure", err)
	}
	if _, err := store.Open(context.Background(), key); !errors.Is(err, establishment.ErrObjectNotFound) {
		t.Fatalf("open after failed write error = %v, want ErrObjectNotFound", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("list local storage directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("failed write left %d temporary or final entries", len(entries))
	}
}

func TestLocalStoreRejectsEscapingKeysAndSymlinks(t *testing.T) {
	store, root := newTestLocalStore(t)
	for _, key := range []establishment.ObjectKey{
		"../outside", establishment.ObjectKey(filepath.Join("/tmp", string(newTestObjectKey(t)))),
		"00000000-0000-0000-0000-000000000000",
	} {
		if err := store.Put(context.Background(), key, bytes.NewReader([]byte("unsafe"))); !errors.Is(err, establishment.ErrInvalidObjectKey) {
			t.Errorf("put with key %q error = %v, want ErrInvalidObjectKey", key, err)
		}
	}

	outsidePath := filepath.Join(t.TempDir(), "outside-object")
	outsideContent := []byte("outside")
	if err := os.WriteFile(outsidePath, outsideContent, 0o600); err != nil {
		t.Fatalf("create outside file: %v", err)
	}
	symlinkKey := newTestObjectKey(t)
	if err := os.Symlink(outsidePath, filepath.Join(root, string(symlinkKey))); err != nil {
		t.Fatalf("create escaping symlink: %v", err)
	}
	if _, err := store.Open(context.Background(), symlinkKey); !errors.Is(err, establishment.ErrStorageFailure) {
		t.Fatalf("open escaping symlink error = %v, want ErrStorageFailure", err)
	}
	if err := store.Put(context.Background(), symlinkKey, bytes.NewReader([]byte("replacement"))); !errors.Is(err, establishment.ErrObjectAlreadyExists) {
		t.Fatalf("put over symlink error = %v, want ErrObjectAlreadyExists", err)
	}
	actualOutside, err := os.ReadFile(outsidePath)
	if err != nil || !bytes.Equal(actualOutside, outsideContent) {
		t.Fatalf("outside file changed: content=%q err=%v", actualOutside, err)
	}
}

func TestLocalStoreConcurrentWritesDoNotOverwrite(t *testing.T) {
	store, _ := newTestLocalStore(t)
	key := newTestObjectKey(t)
	contents := [][]byte{[]byte("first"), []byte("second")}
	results := make(chan error, len(contents))
	var writers sync.WaitGroup
	for _, content := range contents {
		content := content
		writers.Add(1)
		go func() {
			defer writers.Done()
			results <- store.Put(context.Background(), key, bytes.NewReader(content))
		}()
	}
	writers.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, establishment.ErrObjectAlreadyExists):
			conflicts++
		default:
			t.Fatalf("concurrent put error = %v", err)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("concurrent puts: successes=%d conflicts=%d; want one each", successes, conflicts)
	}
	reader, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatalf("open concurrently written object: %v", err)
	}
	defer reader.Close()
	actual, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read concurrently written object: %v", err)
	}
	if !bytes.Equal(actual, contents[0]) && !bytes.Equal(actual, contents[1]) {
		t.Fatalf("concurrent write produced unexpected partial content %q", actual)
	}
}

func TestLocalStoreHonorsCanceledContext(t *testing.T) {
	store, root := newTestLocalStore(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	key := newTestObjectKey(t)
	if err := store.Put(ctx, key, bytes.NewReader([]byte("content"))); !errors.Is(err, establishment.ErrStorageFailure) {
		t.Fatalf("put with canceled context error = %v, want ErrStorageFailure", err)
	}
	if _, err := store.Open(ctx, key); !errors.Is(err, establishment.ErrStorageFailure) {
		t.Fatalf("open with canceled context error = %v, want ErrStorageFailure", err)
	}
	if err := store.Delete(ctx, key); !errors.Is(err, establishment.ErrStorageFailure) {
		t.Fatalf("delete with canceled context error = %v, want ErrStorageFailure", err)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatalf("list local storage directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("canceled operations left %d local entries", len(entries))
	}
}

func TestLocalStorageFailureKeepsRootPathOutOfErrorText(t *testing.T) {
	rootPath := filepath.Join(t.TempDir(), "private-storage-root")
	cause := errors.New("open " + rootPath + ": permission denied")
	failure := storageFailure(cause)
	if !errors.Is(failure, establishment.ErrStorageFailure) || !errors.Is(failure, cause) {
		t.Fatalf("storage failure does not preserve error identity: %v", failure)
	}
	if strings.Contains(failure.Error(), rootPath) {
		t.Fatalf("storage failure exposed root path: %q", failure)
	}
}

func assertObjectContent(t *testing.T, store establishment.ObjectStorage, key establishment.ObjectKey, want []byte) {
	t.Helper()
	reader, err := store.Open(context.Background(), key)
	if err != nil {
		t.Fatalf("open object: %v", err)
	}
	defer reader.Close()
	actual, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("read object: %v", err)
	}
	if !bytes.Equal(actual, want) {
		t.Fatalf("object content = %q, want %q", actual, want)
	}
}

func newTestLocalStore(t *testing.T) (*LocalStore, string) {
	t.Helper()
	root := t.TempDir()
	store, err := NewLocalStore(root)
	if err != nil {
		t.Fatalf("create local store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close local store: %v", err)
		}
	})
	return store, root
}

func newTestObjectKey(t *testing.T) establishment.ObjectKey {
	t.Helper()
	key, err := establishment.NewObjectKey()
	if err != nil {
		t.Fatalf("generate object key: %v", err)
	}
	return key
}

type failingReader struct {
	err error
}

func (reader *failingReader) Read(buffer []byte) (int, error) {
	copy(buffer, "partial")
	return len("partial"), reader.err
}
