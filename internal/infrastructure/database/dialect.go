package db

import (
	"fmt"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func NowExpr(driver string) string {
	switch driver {
	case config.DatabaseDriverSQLite:
		return "datetime('now')"
	case config.DatabaseDriverMySQL:
		return "CURRENT_TIMESTAMP(3)"
	default:
		return "datetime('now')"
	}
}

func DurationMillisExpr(driver string, startColumn string) string {
	switch driver {
	case config.DatabaseDriverSQLite:
		return fmt.Sprintf("CAST((julianday(datetime('now')) - julianday(%s)) * 86400000 AS INTEGER)", startColumn)
	case config.DatabaseDriverMySQL:
		return fmt.Sprintf("TIMESTAMPDIFF(MICROSECOND, %s, CURRENT_TIMESTAMP(3)) DIV 1000", startColumn)
	default:
		return fmt.Sprintf("CAST((julianday(datetime('now')) - julianday(%s)) * 86400000 AS INTEGER)", startColumn)
	}
}

func QuoteIdent(driver string, name string) string {
	switch driver {
	case config.DatabaseDriverMySQL:
		return "`" + name + "`"
	default:
		return name
	}
}
