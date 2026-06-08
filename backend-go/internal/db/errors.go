package db

import "database/sql"

func IsNoRows(err error) bool {
	return err == sql.ErrNoRows
}
