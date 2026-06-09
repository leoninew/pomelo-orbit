package db

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"sort"
	"time"

	"github.com/jmoiron/sqlx"

	"backend/internal/config"
	"backend/internal/migrations"
	"backend/internal/templatex"
)

type Migrator struct {
	db      *sqlx.DB
	driver  string
	context map[string]any
}

type MigrationStatus struct {
	Filename string
	Checksum string
	Applied  bool
}

func NewMigrator(db *sqlx.DB, driver string) Migrator {
	return Migrator{db: db, driver: driver, context: map[string]any{}}
}

func NewMigratorWithContext(db *sqlx.DB, driver string, context map[string]any) Migrator {
	return Migrator{db: db, driver: driver, context: context}
}

func (m Migrator) Up() error {
	if err := m.ensureHistoryTable(); err != nil {
		return err
	}

	files, err := migrationFiles(m.driver)
	if err != nil {
		return err
	}

	for _, file := range files {
		content, err := m.readMigration(file)
		if err != nil {
			return err
		}
		fileChecksum := checksum([]byte(content))
		applied, err := m.applied(file, fileChecksum)
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		if err := m.apply(file, fileChecksum, content); err != nil {
			return fmt.Errorf("apply migration %s: %w", file, err)
		}
	}
	return nil
}

func (m Migrator) Status() ([]MigrationStatus, error) {
	if err := m.ensureHistoryTable(); err != nil {
		return nil, err
	}

	files, err := migrationFiles(m.driver)
	if err != nil {
		return nil, err
	}

	statuses := make([]MigrationStatus, 0, len(files))
	for _, file := range files {
		content, err := m.readMigration(file)
		if err != nil {
			return nil, err
		}
		fileChecksum := checksum([]byte(content))
		applied, err := m.applied(file, fileChecksum)
		if err != nil {
			return nil, err
		}
		statuses = append(statuses, MigrationStatus{Filename: file, Checksum: fileChecksum, Applied: applied})
	}
	return statuses, nil
}

func migrationFiles(driver string) ([]string, error) {
	files, err := fs.Glob(migrations.Files, driver+"/*.sql")
	if err != nil {
		return nil, fmt.Errorf("list migrations: %w", err)
	}
	sort.Strings(files)
	return files, nil
}

func (m Migrator) readMigration(file string) (string, error) {
	content, err := migrations.Files.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read migration %s: %w", file, err)
	}
	return renderMigrationContent(file, string(content), m.context)
}

func renderMigrationContent(filename string, content string, context map[string]any) (string, error) {
	if len(context) == 0 {
		return content, nil
	}
	rendered, err := templatex.Render(content, context)
	if err != nil {
		return "", fmt.Errorf("render migration %s: %w", filename, err)
	}
	return rendered, nil
}

func (m Migrator) ensureHistoryTable() error {
	statement := `CREATE TABLE IF NOT EXISTS __migration_history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		filename TEXT NOT NULL UNIQUE,
		checksum TEXT NOT NULL,
		executed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
		execution_time_ms INTEGER NOT NULL
	)`
	if m.driver == config.DatabaseDriverMySQL {
		statement = `CREATE TABLE IF NOT EXISTS __migration_history (
			id BIGINT AUTO_INCREMENT PRIMARY KEY,
			filename VARCHAR(255) NOT NULL UNIQUE,
			checksum VARCHAR(64) NOT NULL,
			executed_at DATETIME(3) NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
			execution_time_ms BIGINT NOT NULL
		)`
	}
	_, err := m.db.Exec(statement)
	if err != nil {
		return fmt.Errorf("ensure migration history: %w", err)
	}
	return nil
}

func (m Migrator) applied(filename string, fileChecksum string) (bool, error) {
	var stored string
	err := m.db.Get(&stored, "SELECT checksum FROM __migration_history WHERE filename = ?", filename)
	if err == nil {
		if stored != fileChecksum {
			return false, fmt.Errorf("checksum mismatch for migration %s: stored=%s current=%s", filename, stored, fileChecksum)
		}
		return true, nil
	}
	if IsNoRows(err) {
		return false, nil
	}
	return false, fmt.Errorf("check migration %s: %w", filename, err)
}

func (m Migrator) apply(filename string, fileChecksum string, sql string) error {
	tx, err := m.db.Beginx()
	if err != nil {
		return fmt.Errorf("begin migration transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	startedAt := time.Now()
	if _, err := tx.Exec(sql); err != nil {
		return err
	}
	durationMS := int(time.Since(startedAt).Milliseconds())
	if _, err := tx.Exec(
		"INSERT INTO __migration_history (filename, checksum, execution_time_ms) VALUES (?, ?, ?)",
		filename,
		fileChecksum,
		durationMS,
	); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration: %w", err)
	}
	return nil
}

func checksum(content []byte) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
