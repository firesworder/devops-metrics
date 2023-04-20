package storage

import (
	"context"
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
	// Этот метод вызывается при инициализации сервера, поэтому использую общий контекст
	ctx := context.Background()

	db := SQLStorage{}
	err := db.OpenDBConnection(DSN)
	if err != nil {
		return nil, err
	}
	err = db.CreateTableIfNotExist(ctx)
	if err != nil {
		return nil, err
	}
	if err = db.initStmts(ctx); err != nil {
		return nil, err
	}
	return &db, nil
}

func (db *SQLStorage) initStmts(ctx context.Context) (err error) {
	insertStmt, err = db.Connection.PrepareContext(
		ctx,
		"INSERT INTO metrics(m_name, m_value, m_type) VALUES($1, $2, $3)",
	)
	if err != nil {
		return
	}

	updateStmt, err = db.Connection.PrepareContext(
		ctx,
		"UPDATE metrics SET m_value = $2, m_type = $3 WHERE m_name = $1",
	)
	if err != nil {
		return err
	}

	deleteStmt, err = db.Connection.PrepareContext(ctx, "DELETE FROM metrics WHERE m_name = $1")
	if err != nil {
		return err
	}

	selectMetricStmt, err = db.Connection.PrepareContext(
		ctx,
		"SELECT m_name, m_value, m_type FROM metrics WHERE m_name = $1 LIMIT 1",
	)
	if err != nil {
		return err
	}

	selectAllStmt, err = db.Connection.PrepareContext(ctx, "SELECT m_name, m_value, m_type FROM metrics")
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

func (db *SQLStorage) CreateTableIfNotExist(ctx context.Context) (err error) {
	_, err = db.Connection.ExecContext(
		ctx,
		`CREATE TABLE IF NOT EXISTS metrics
		(
			id      SERIAL PRIMARY KEY,
			m_name  VARCHAR(50) UNIQUE,
			m_value VARCHAR(50),
			m_type  VARCHAR(20)
		);`,
	)
	if err != nil {
		return
	}
	return nil
}

// MetricRepository реализация

func (db *SQLStorage) AddMetric(ctx context.Context, metric Metric) (err error) {
	mN, mV, mT := metric.GetMetricParamsString()
	_, err = insertStmt.ExecContext(ctx, mN, mV, mT)
	if err != nil {
		return
	}
	return
}

func (db *SQLStorage) UpdateMetric(ctx context.Context, metric Metric) (err error) {
	dbMetric, isOk := db.GetMetric(ctx, metric.Name)
	if !isOk {
		return fmt.Errorf("metric to update was not found")
	}
	err = dbMetric.Update(metric.Value)
	if err != nil {
		return err
	}

	mN, mV, mT := metric.GetMetricParamsString()
	result, err := updateStmt.ExecContext(ctx, mN, mV, mT)
	if err != nil {
		return err
	}
	rAff, err := result.RowsAffected()
	if rAff == 0 {
		return fmt.Errorf("metric to update was not found")
	}
	return
}

func (db *SQLStorage) DeleteMetric(ctx context.Context, metric Metric) (err error) {
	result, err := deleteStmt.ExecContext(ctx, metric.Name)
	if err != nil {
		return err
	}
	rAff, err := result.RowsAffected()
	if rAff == 0 {
		return fmt.Errorf("metric to delete was not found")
	}
	return
}

func (db *SQLStorage) IsMetricInStorage(ctx context.Context, metric Metric) bool {
	result, err := selectMetricStmt.ExecContext(ctx, metric.Name)
	if err != nil {
		return false
	}
	rAff, err := result.RowsAffected()
	return rAff != 0 || err != nil
}

func (db *SQLStorage) UpdateOrAddMetric(ctx context.Context, metric Metric) (err error) {
	if db.IsMetricInStorage(ctx, metric) {
		err = db.UpdateMetric(ctx, metric)
	} else {
		err = db.AddMetric(ctx, metric)
	}
	return
}

func (db *SQLStorage) GetAll(ctx context.Context) (result map[string]Metric) {
	result = map[string]Metric{}
	rows, err := selectAllStmt.QueryContext(ctx)
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

func (db *SQLStorage) GetMetric(ctx context.Context, name string) (metric Metric, isOk bool) {
	rows, err := selectMetricStmt.QueryContext(ctx, name)
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

func (db *SQLStorage) BatchUpdate(ctx context.Context, metrics []Metric) (err error) {
	existedMetrics := db.GetAll(ctx)
	tx, err := db.Connection.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()
	txUpdateStmt := tx.StmtContext(ctx, updateStmt)
	txInsertStmt := tx.StmtContext(ctx, insertStmt)

	metricsUpdate := map[string]Metric{}
	for _, metric := range metrics {
		if metricUpdate, ok := metricsUpdate[metric.Name]; ok {
			err = metricUpdate.Update(metric.Value)
			if err != nil {
				return err
			}
			metricsUpdate[metric.Name] = metricUpdate
		} else {
			metricsUpdate[metric.Name] = metric
		}
	}

	var mN, mV, mT string
	for mName, metric := range metricsUpdate {
		if existedMetric, ok := existedMetrics[mName]; ok {
			if err = existedMetric.Update(metric.Value); err != nil {
				return err
			}
			mN, mV, mT = existedMetric.GetMetricParamsString()
			if _, err = txUpdateStmt.ExecContext(ctx, mN, mV, mT); err != nil {
				return err
			}
		} else {
			mN, mV, mT = metric.GetMetricParamsString()
			if _, err = txInsertStmt.ExecContext(ctx, mN, mV, mT); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
