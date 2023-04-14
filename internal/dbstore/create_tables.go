package dbstore

import (
	"database/sql"
	"github.com/firesworder/devopsmetrics/internal"
	_ "github.com/jackc/pgx/v5/stdlib"
	"os"
)

type DBStore struct {
	Connection internal.DBStorage
}

func NewDBStore(DSN string) (*DBStore, error) {
	db := DBStore{}
	err := db.OpenDBConnection(DSN)
	if err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *DBStore) OpenDBConnection(DSN string) error {
	var err error
	db.Connection, err = sql.Open("pgx", DSN)
	if err != nil {
		return err
	}
	return nil
}

func (db *DBStore) CreateTableIfNotExist() error {
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
