package migrationfiles

import "embed"

// Files contains only automatic schema migrations.
//go:embed migration/sqlite/*.sql migration/mysql/*.sql
var Files embed.FS

// DataFiles contains business data migrations. They are applied by the
// application migration flow after MigrateUp completes.
//go:embed migration/data/sqlite/*.sql migration/data/mysql/*.sql
var DataFiles embed.FS
