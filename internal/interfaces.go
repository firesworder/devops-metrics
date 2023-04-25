package internal

import (
	"database/sql"
	"io"
)

type DBStorage interface {
	Ping() error
	io.Closer
	Query(query string, args ...any) (*sql.Rows, error)
	Exec(query string, args ...any) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
}

type DBStmt interface {
	Query(args ...any) (*sql.Rows, error)
	Exec(args ...any) (sql.Result, error)
}

type DBResult interface {
	sql.Result
}

type DBRows interface {
	Next() bool
	Scan(args ...any) error
}
