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
}
