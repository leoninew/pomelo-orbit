package repository

import (
	"testing"

	"github.com/oklog/ulid/v2"
)

func TestNewId(t *testing.T) {
	id := NewId()
	if len(id) != 26 {
		t.Fatalf("expected 26-char id, got %q", id)
	}
	if _, err := ulid.ParseStrict(id); err != nil {
		t.Fatalf("expected ULID, got %q: %v", id, err)
	}
	if id == NewId() {
		t.Fatal("expected unique ids")
	}
}
