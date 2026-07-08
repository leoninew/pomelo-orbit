package repository

import "testing"

func TestNewId(t *testing.T) {
	id := NewId()
	if len(id) != 26 {
		t.Fatalf("expected 26-char id, got %q", id)
	}
	if id == NewId() {
		t.Fatal("expected unique ids")
	}
}
