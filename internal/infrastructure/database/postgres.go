package db

import (
	"context"
	"database/sql/driver"
	"fmt"
	"strings"
)

type postgresPlaceholderConnector struct {
	driver.Connector
}

func (c postgresPlaceholderConnector) Connect(ctx context.Context) (driver.Conn, error) {
	connection, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}
	return &postgresPlaceholderConn{Conn: connection}, nil
}

type postgresPlaceholderConn struct {
	driver.Conn
}

func (c *postgresPlaceholderConn) Prepare(query string) (driver.Stmt, error) {
	return c.Conn.Prepare(rebindPostgresPlaceholders(query))
}

func (c *postgresPlaceholderConn) PrepareContext(ctx context.Context, query string) (driver.Stmt, error) {
	if preparer, ok := c.Conn.(driver.ConnPrepareContext); ok {
		return preparer.PrepareContext(ctx, rebindPostgresPlaceholders(query))
	}
	return c.Prepare(query)
}

func (c *postgresPlaceholderConn) ExecContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Result, error) {
	if execer, ok := c.Conn.(driver.ExecerContext); ok {
		return execer.ExecContext(ctx, rebindPostgresPlaceholders(query), args)
	}
	return nil, driver.ErrSkip
}

func (c *postgresPlaceholderConn) QueryContext(ctx context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	if queryer, ok := c.Conn.(driver.QueryerContext); ok {
		return queryer.QueryContext(ctx, rebindPostgresPlaceholders(query), args)
	}
	return nil, driver.ErrSkip
}

func (c *postgresPlaceholderConn) BeginTx(ctx context.Context, opts driver.TxOptions) (driver.Tx, error) {
	if beginner, ok := c.Conn.(driver.ConnBeginTx); ok {
		return beginner.BeginTx(ctx, opts)
	}
	return nil, driver.ErrSkip
}

func (c *postgresPlaceholderConn) Ping(ctx context.Context) error {
	if pinger, ok := c.Conn.(driver.Pinger); ok {
		return pinger.Ping(ctx)
	}
	return driver.ErrSkip
}

func rebindPostgresPlaceholders(query string) string {
	var builder strings.Builder
	builder.Grow(len(query))

	const (
		normal = iota
		singleQuoted
		doubleQuoted
		backtickQuoted
		lineComment
		blockComment
	)

	state := normal
	parameterIndex := 0
	for index := 0; index < len(query); index++ {
		character := query[index]
		switch state {
		case normal:
			switch character {
			case '\'':
				builder.WriteByte(character)
				state = singleQuoted
			case '"':
				builder.WriteByte(character)
				state = doubleQuoted
			case 96:
				builder.WriteByte('"')
				state = backtickQuoted
			case '-':
				if index+1 < len(query) && query[index+1] == '-' {
					builder.WriteString("--")
					index++
					state = lineComment
					continue
				}
				builder.WriteByte(character)
			case '/':
				if index+1 < len(query) && query[index+1] == '*' {
					builder.WriteString("/*")
					index++
					state = blockComment
					continue
				}
				builder.WriteByte(character)
			case '?':
				parameterIndex++
				_, _ = fmt.Fprintf(&builder, "$%d", parameterIndex)
			default:
				builder.WriteByte(character)
			}
		case singleQuoted:
			builder.WriteByte(character)
			if character == '\\' && index+1 < len(query) {
				index++
				builder.WriteByte(query[index])
				continue
			}
			if character == '\'' {
				if index+1 < len(query) && query[index+1] == '\'' {
					index++
					builder.WriteByte(query[index])
					continue
				}
				state = normal
			}
		case doubleQuoted:
			builder.WriteByte(character)
			if character == '"' {
				if index+1 < len(query) && query[index+1] == '"' {
					index++
					builder.WriteByte(query[index])
					continue
				}
				state = normal
			}
		case backtickQuoted:
			if character == 96 {
				if index+1 < len(query) && query[index+1] == 96 {
					index++
					builder.WriteString(`""`)
					continue
				}
				builder.WriteByte('"')
				state = normal
				continue
			}
			builder.WriteByte(character)
		case lineComment:
			builder.WriteByte(character)
			if character == '\n' {
				state = normal
			}
		case blockComment:
			builder.WriteByte(character)
			if character == '*' && index+1 < len(query) && query[index+1] == '/' {
				index++
				builder.WriteByte(query[index])
				state = normal
			}
		}
	}
	return builder.String()
}
