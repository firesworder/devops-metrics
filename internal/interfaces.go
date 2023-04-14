package internal

import "io"

type DBStorage interface {
	Ping() error
	io.Closer
}
