package testseed

import (
	"database/sql"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func ApplySQLitePipelineDemo(t testing.TB, database *sql.DB) {
	t.Helper()
	applySQLiteExport(t, database, "000041_pipeline-demo.sqlite.sql")
}

func applySQLiteExport(t testing.TB, database *sql.DB, name string) {
	t.Helper()
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test seed package")
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(path), "..", "..", "..", "data", "exports", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if _, err := database.Exec(string(contents)); err != nil {
		t.Fatalf("apply %s: %v", name, err)
	}
}
