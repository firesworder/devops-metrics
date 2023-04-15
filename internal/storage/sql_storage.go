package storage

import (
	"database/sql"
	"github.com/firesworder/devopsmetrics/internal"
	_ "github.com/jackc/pgx/v5/stdlib"
	"os"
)

type SqlStorage struct {
	Connection internal.DBStorage
}

func NewSqlStorage(DSN string) (*SqlStorage, error) {
	db := SqlStorage{}
	err := db.OpenDBConnection(DSN)
	if err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *SqlStorage) OpenDBConnection(DSN string) error {
	var err error
	db.Connection, err = sql.Open("pgx", DSN)
	if err != nil {
		return err
	}
	return nil
}

func (db *SqlStorage) CreateTableIfNotExist() error {
	sqlScript, err := os.ReadFile("metrics.sql")
	if err != nil {
		return err
	}
	_, err = db.Connection.Exec(string(sqlScript))
	if err != nil {
		return err
	}
	return nil
}
