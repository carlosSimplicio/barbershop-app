package establishment

import (
	"testing"

	"github.com/google/uuid"
)

func TestNewObjectKeyCreatesOpaqueRandomUUIDv4(t *testing.T) {
	first, err := NewObjectKey()
	if err != nil {
		t.Fatalf("create first object key: %v", err)
	}
	second, err := NewObjectKey()
	if err != nil {
		t.Fatalf("create second object key: %v", err)
	}
	for _, key := range []ObjectKey{first, second} {
		parsed, err := uuid.Parse(string(key))
		if err != nil || parsed.Version() != uuid.Version(4) || parsed.String() != string(key) {
			t.Errorf("object key %q is not a canonical UUID v4: %v", key, err)
		}
	}
	if first == second {
		t.Fatal("two generated object keys collided")
	}
}
