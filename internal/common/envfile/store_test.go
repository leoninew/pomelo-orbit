package envfile

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
)

func TestCodecPreservesLiteralValues(t *testing.T) {
	values := map[string]string{
		"EMPTY": "", "ZERO": "0012", "BOOL": "false", "DOLLAR": "${SECRET} $HOME",
		"QUOTES": "'quoted' and \"double\"", "PATH": " C:\\folder\\ ", "TRAILING": "C:\\folder\\",
		"MULTILINE": "line one\nline two\r\n", "COMMENT": "value # literal",
		"ESCAPES": "\x01\t\b\f", "UNICODE": "\u4e2d\u6587 \U0001f600\u2028",
	}
	content, err := Encode(values)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := Decode(content)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(values, decoded) {
		t.Fatalf("round-trip changed values: %#v", decoded)
	}
	decoded, err = Decode([]byte("# comment\nexport KEY=\"value\" # comment\nLITERAL='${HOME}'\nRAW=${HOME}\n"))
	if err != nil || decoded["KEY"] != "value" || decoded["LITERAL"] != "${HOME}" || decoded["RAW"] != "${HOME}" {
		t.Fatalf("decode = %#v, %v", decoded, err)
	}
}

func TestStoreMutationsPreserveInodeAndUnrelatedValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overwrite.env")
	testStoreMutations(t, path)
}

func TestStoreRejectsOversizedAssignmentWithoutChangingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overwrite.env")
	original := "VALUE=old\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewStore(path).Mutate(context.Background(), func(values map[string]string) error {
		values["VALUE"] = strings.Repeat("x", 64*1024)
		return nil
	})
	if err == nil {
		t.Fatal("expected oversized assignment to be rejected")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != original {
		t.Fatalf("original changed: %q, %v", content, err)
	}
}

func TestStoreMountedTarget(t *testing.T) {
	path := os.Getenv("POMELO_TEST_MOUNTED_ENV_PATH")
	if path == "" {
		t.Skip("requires an isolated writable file bind mount")
	}
	content, err := os.ReadFile(path)
	if err != nil || len(content) != 0 {
		t.Fatalf("mounted fixture must be an existing empty file: %v", err)
	}
	testStoreMutations(t, path)
}

func testStoreMutations(t *testing.T, path string) {
	t.Helper()
	if err := os.WriteFile(path, []byte("FIRST=old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	before, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(path)
	_, err = store.Mutate(context.Background(), func(values map[string]string) error { values["SECOND"] = "new"; return nil })
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !os.SameFile(before, after) {
		t.Fatal("target inode was replaced")
	}
	values, err := store.Load(context.Background())
	if err != nil || values["FIRST"] != "old" || values["SECOND"] != "new" {
		t.Fatalf("values = %#v, %v", values, err)
	}
	_, err = store.Mutate(context.Background(), func(values map[string]string) error { delete(values, "SECOND"); return nil })
	if err != nil {
		t.Fatal(err)
	}
	values, err = store.Load(context.Background())
	if err != nil || len(values) != 1 || values["FIRST"] != "old" {
		t.Fatalf("reset changed unrelated values: %#v, %v", values, err)
	}
}

func TestStoreRecoversInterruptedWriteAndRetainsCommittedWrite(t *testing.T) {
	for _, phase := range []string{"prepared", "committed"} {
		t.Run(phase, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "overwrite.env")
			store := NewStore(path)
			if err := os.MkdirAll(store.directory, 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(store.record(phase), []byte("VALUE=old\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			content := "BROKEN=\"unterminated"
			want := "old"
			if phase == "committed" {
				content, want = "VALUE=new\n", "new"
			}
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			values, err := store.Load(context.Background())
			if err != nil || values["VALUE"] != want {
				t.Fatalf("recovery = %#v, %v", values, err)
			}
			if _, err := os.Stat(store.record(phase)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("record retained: %v", err)
			}
		})
	}
}

func TestStoreRejectedMutationRetainsOriginal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overwrite.env")
	original := "VALUE=old\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewStore(path).Mutate(context.Background(), func(values map[string]string) error { values["VALUE"] = "new"; return errors.New("invalid candidate") })
	if err == nil {
		t.Fatal("expected validation failure")
	}
	content, err := os.ReadFile(path)
	if err != nil || string(content) != original {
		t.Fatalf("original changed: %q, %v", content, err)
	}
}

func TestStoreConcurrentProcessesDoNotLoseValues(t *testing.T) {
	path := filepath.Join(t.TempDir(), "overwrite.env")
	var wait sync.WaitGroup
	for _, key := range []string{"FIRST", "SECOND"} {
		wait.Go(func() {
			command := exec.Command(os.Args[0], "-test.run=^TestStoreProcessHelper$")
			command.Env = append(os.Environ(), "POMELO_TEST_ENV_PATH="+path, "POMELO_TEST_ENV_KEY="+key)
			if output, err := command.CombinedOutput(); err != nil {
				t.Errorf("child failed: %v: %s", err, output)
			}
		})
	}
	wait.Wait()
	values, err := NewStore(path).Load(context.Background())
	if err != nil || values["FIRST"] != "value" || values["SECOND"] != "value" {
		t.Fatalf("concurrent values = %#v, %v", values, err)
	}
}

func TestStoreProcessHelper(t *testing.T) {
	path := os.Getenv("POMELO_TEST_ENV_PATH")
	if path == "" {
		return
	}
	_, err := NewStore(path).Mutate(context.Background(), func(values map[string]string) error { values[os.Getenv("POMELO_TEST_ENV_KEY")] = "value"; return nil })
	if err != nil {
		t.Fatal(err)
	}
}
