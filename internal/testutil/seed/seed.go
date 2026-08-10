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
	_, err := database.Exec(`
        INSERT OR IGNORE INTO repository (id, name, code, repository_url, project_id)
        VALUES (
            '01KNNRBH52BQJYT9487B2H8N62',
            'Pipeline Demo',
            'pipeline-demo',
            'https://example.invalid/pipeline-demo.git',
            '01KRRKK0K3T519ZQZES3M4QA9Z'
        )
    `)
	if err != nil {
		t.Fatalf("seed pipeline demo repository: %v", err)
	}
}

func ApplySQLiteSystemSeed(t testing.TB, database *sql.DB) {
	t.Helper()
	applySQLiteDataMigration(t, database, "000029_seed_system.up.sql")
}

func applySQLiteDataMigration(t testing.TB, database *sql.DB, name string) {
	t.Helper()
	_, path, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate test seed package")
	}
	contents, err := os.ReadFile(filepath.Join(filepath.Dir(path), "..", "..", "..", "sql", "migration", "data", "sqlite", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	if _, err := database.Exec(string(contents)); err != nil {
		t.Fatalf("apply %s: %v", name, err)
	}
}
