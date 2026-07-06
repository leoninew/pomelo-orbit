package db

import (
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func TestOpenSQLiteConfiguresPragmas(t *testing.T) {
	database, err := openSQLite(config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "pomelo.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	var busyTimeout int
	if err := database.Get(&busyTimeout, "PRAGMA busy_timeout"); err != nil {
		t.Fatal(err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("expected busy_timeout 5000, got %d", busyTimeout)
	}

	var journalMode string
	if err := database.Get(&journalMode, "PRAGMA journal_mode"); err != nil {
		t.Fatal(err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("expected journal_mode wal, got %q", journalMode)
	}

	var foreignKeys int
	if err := database.Get(&foreignKeys, "PRAGMA foreign_keys"); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign_keys on, got %d", foreignKeys)
	}
}
