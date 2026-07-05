package migrationfiles

import "embed"

//go:embed migration/sqlite/*.sql migration/mysql/*.sql
var Files embed.FS
