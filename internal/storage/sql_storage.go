package storage

import (
	"database/sql"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
	_ "github.com/jackc/pgx/v5/stdlib"
	"strconv"
)

var insertStmt, updateStmt, deleteStmt *sql.Stmt
var selectMetricStmt, selectAllStmt *sql.Stmt

type SQLStorage struct {
	Connection *sql.DB
}

func NewSQLStorage(DSN string) (*SQLStorage, error) {
	db := SQLStorage{}
	err := db.OpenDBConnection(DSN)
	if err != nil {
		return nil, err
	}
	err = db.CreateTableIfNotExist()
	if err != nil {
		return nil, err
	}
	if err = db.initStmts(); err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *SQLStorage) initStmts() (err error) {
	insertStmt, err = db.Connection.Prepare("INSERT INTO metrics(m_name, m_value, m_type) VALUES($1, $2, $3)")
	if err != nil {
		return
	}

	updateStmt, err = db.Connection.Prepare("UPDATE metrics SET m_value = $2, m_type = $3 WHERE m_name = $1")
	if err != nil {
		return err
	}

	deleteStmt, err = db.Connection.Prepare("DELETE FROM metrics WHERE m_name = $1")
	if err != nil {
		return err
	}

	selectMetricStmt, err = db.Connection.Prepare(
		"SELECT m_name, m_value, m_type FROM metrics WHERE m_name = $1 LIMIT 1")
	if err != nil {
		return err
	}

	selectAllStmt, err = db.Connection.Prepare("SELECT m_name, m_value, m_type FROM metrics")
	if err != nil {
		return err
	}
	return
}

func (db *SQLStorage) OpenDBConnection(DSN string) error {
	var err error
	db.Connection, err = sql.Open("pgx", DSN)
	if err != nil {
		return err
	}
	return nil
}

func (db *SQLStorage) CreateTableIfNotExist() (err error) {
	_, err = db.Connection.Exec(`
		CREATE TABLE IF NOT EXISTS metrics
		(
			id      SERIAL PRIMARY KEY,
			m_name  VARCHAR(50) UNIQUE,
			m_value VARCHAR(50),
			m_type  VARCHAR(20)
		);
`)
	if err != nil {
		return
	}
	return nil
}

// MetricRepository реализация

func (db *SQLStorage) AddMetric(metric Metric) (err error) {
	_, err = insertStmt.Exec(metric.GetMetricParamsString())
	if err != nil {
		return
	}
	return
}

func (db *SQLStorage) UpdateMetric(metric Metric) (err error) {
	dbMetric, isOk := db.GetMetric(metric.Name)
	if !isOk {
		return fmt.Errorf("metric to update was not found")
	}
	err = dbMetric.Update(metric.Value)
	if err != nil {
		return err
	}

	result, err := updateStmt.Exec(dbMetric.GetMetricParamsString())
	if err != nil {
		return err
	}
	rAff, err := result.RowsAffected()
	if rAff == 0 {
		return fmt.Errorf("metric to update was not found")
	}
	return
}

func (db *SQLStorage) DeleteMetric(metric Metric) (err error) {
	result, err := deleteStmt.Exec(metric.Name)
	if err != nil {
		return err
	}
	rAff, err := result.RowsAffected()
	if rAff == 0 {
		return fmt.Errorf("metric to delete was not found")
	}
	return
}

func (db *SQLStorage) IsMetricInStorage(metric Metric) bool {
	result, err := selectMetricStmt.Exec(metric.Name)
	if err != nil {
		return false
	}
	rAff, err := result.RowsAffected()
	return rAff != 0 || err != nil
}

func (db *SQLStorage) UpdateOrAddMetric(metric Metric) (err error) {
	if db.IsMetricInStorage(metric) {
		err = db.UpdateMetric(metric)
	} else {
		err = db.AddMetric(metric)
	}
	return
}

func (db *SQLStorage) GetAll() (result map[string]Metric) {
	result = map[string]Metric{}
	rows, err := selectAllStmt.Query()
	if err != nil {
		return
	}

	var mN, mV, mT string
	var mValue interface{}
	var metric *Metric
	for rows.Next() {
		// сбрасываю значения переменных в начале итерации
		mN, mV, mT, mValue, metric = "", "", "", nil, nil

		err = rows.Scan(&mN, &mV, &mT)
		if err != nil {
			return
		}

		switch mT {
		case internal.GaugeTypeName:
			mValue, err = strconv.ParseFloat(mV, 64)
			if err != nil {
				return
			}
		case internal.CounterTypeName:
			mValue, err = strconv.ParseInt(mV, 10, 64)
			if err != nil {
				return
			}
		}
		metric, err = NewMetric(mN, mT, mValue)
		if err != nil {
			return
		}
		result[mN] = *metric
	}

	// проверяем на ошибки
	err = rows.Err()
	if err != nil {
		return
	}
	return
}

func (db *SQLStorage) GetMetric(name string) (metric Metric, isOk bool) {
	rows, err := selectMetricStmt.Query(name)
	if err != nil {
		return
	}

	var mN, mV, mT string
	var mValue interface{}
	rows.Next()
	err = rows.Scan(&mN, &mV, &mT)
	if err != nil {
		return
	}
	switch mT {
	case internal.GaugeTypeName:
		mValue, err = strconv.ParseFloat(mV, 64)
		if err != nil {
			return
		}
	case internal.CounterTypeName:
		mValue, err = strconv.ParseInt(mV, 10, 64)
		if err != nil {
			return
		}
	}
	m, err := NewMetric(mN, mT, mValue)
	if err != nil {
		return
	}

	err = rows.Err()
	if err != nil {
		return
	}
	return *m, true
}

func (db *SQLStorage) BatchUpdate(metrics map[string]Metric) (err error) {
	existedMetrics := db.GetAll()
	tx, err := db.Connection.Begin()
	if err != nil {
		return
	}
	defer tx.Rollback()
	txUpdateStmt := tx.Stmt(updateStmt)
	txInsertStmt := tx.Stmt(insertStmt)

	for mName, metric := range metrics {
		if existedMetric, ok := existedMetrics[mName]; ok {
			if err = existedMetric.Update(metric.Value); err != nil {
				return err
			}
			if _, err = txUpdateStmt.Exec(existedMetric.GetMetricParamsString()); err != nil {
				return err
			}
		} else {
			if _, err = txInsertStmt.Exec(metric.GetMetricParamsString()); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
