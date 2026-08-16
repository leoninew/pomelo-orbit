package migrationfiles

import "embed"

// Files contains only automatic schema migrations.
//
//go:embed migration/sqlite/*.sql migration/mysql/*.sql
var Files embed.FS
