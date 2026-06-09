package migrations

import (
	"embed"
)

//go:embed sqlite/*.sql mysql/*.sql
var Files embed.FS
