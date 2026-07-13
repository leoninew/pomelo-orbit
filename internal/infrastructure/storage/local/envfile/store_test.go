package envfile

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestStoreLoadMissingFileReturnsEmptyMap(t *testing.T) {
	store := NewStore(filepath.Join(t.TempDir(), "missing", ".env"))

	values, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(values, map[string]string{}) {
		t.Fatalf("unexpected values: %#v", values)
	}
}

func TestStoreLoadParsesEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	content := "\n# comment\nMALFORMED\n FIRST = one=two \nQUOTED_DOUBLE = \"value\"\nQUOTED_SINGLE='other'\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	store := NewStore(path)

	values, err := store.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"FIRST":         "one=two",
		"QUOTED_DOUBLE": "value",
		"QUOTED_SINGLE": "other",
	}
	if !reflect.DeepEqual(values, want) {
		t.Fatalf("unexpected values: got %#v, want %#v", values, want)
	}
}

func TestStoreSetAndDeletePersistSortedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", ".env")
	store := NewStore(path)
	ctx := context.Background()

	if err := store.Set(ctx, map[string]string{"ZETA": "last", "ALPHA": "first"}); err != nil {
		t.Fatal(err)
	}
	if err := store.Set(ctx, map[string]string{"MIDDLE": "middle", "ALPHA": "updated"}); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "ALPHA=updated\nMIDDLE=middle\nZETA=last\n" {
		t.Fatalf("unexpected content: %q", content)
	}

	if err := store.Delete(ctx, []string{"MIDDLE", "ZETA"}); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "ALPHA=updated\n" {
		t.Fatalf("unexpected content after delete: %q", content)
	}

	if err := store.Delete(ctx, []string{"ALPHA"}); err != nil {
		t.Fatal(err)
	}
	content, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "" {
		t.Fatalf("unexpected empty content: %q", content)
	}
}
