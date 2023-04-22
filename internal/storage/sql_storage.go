package storage

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
	_ "github.com/jackc/pgx/v5/stdlib"
	"strconv"
)

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
	return &db, nil
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
	_, err = db.Connection.ExecContext(ctx,
		"INSERT INTO metrics(m_name, m_value, m_type) VALUES($1, $2, $3)", mN, mV, mT)
	if err != nil {
		return
	}
	return
}

func (db *SQLStorage) UpdateMetric(ctx context.Context, metric Metric) (err error) {
	dbMetric, err := db.GetMetric(ctx, metric.Name)
	if err != nil {
		return
	}
	err = dbMetric.Update(metric.Value)
	if err != nil {
		return
	}

	mN, mV, mT := dbMetric.GetMetricParamsString()
	result, err := db.Connection.ExecContext(ctx,
		"UPDATE metrics SET m_value = $2, m_type = $3 WHERE m_name = $1", mN, mV, mT)
	if err != nil {
		return
	}
	rAff, err := result.RowsAffected()
	if rAff == 0 {
		return fmt.Errorf("metric to update was not found")
	}
	return
}

func (db *SQLStorage) DeleteMetric(ctx context.Context, metric Metric) (err error) {
	result, err := db.Connection.ExecContext(ctx,
		"DELETE FROM metrics WHERE m_name = $1", metric.Name)
	if err != nil {
		return err
	}
	rAff, err := result.RowsAffected()
	if rAff == 0 {
		return ErrMetricNotFound
	}
	return
}

func (db *SQLStorage) IsMetricInStorage(ctx context.Context, metric Metric) (isExist bool, err error) {
	result, err := db.Connection.ExecContext(ctx,
		"SELECT m_name, m_value, m_type FROM metrics WHERE m_name = $1 LIMIT 1", metric.Name)
	if err != nil {
		return
	}
	rAff, err := result.RowsAffected()
	if err != nil {
		return
	}
	return rAff != 0, nil
}

func (db *SQLStorage) UpdateOrAddMetric(ctx context.Context, metric Metric) (err error) {
	mInStorage, err := db.IsMetricInStorage(ctx, metric)
	if err != nil {
		return
	}

	if mInStorage {
		err = db.UpdateMetric(ctx, metric)
	} else {
		err = db.AddMetric(ctx, metric)
	}
	return
}

func (db *SQLStorage) GetAll(ctx context.Context) (result map[string]Metric, err error) {
	result = map[string]Metric{}
	rows, err := db.Connection.QueryContext(ctx, "SELECT m_name, m_value, m_type FROM metrics")
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

func (db *SQLStorage) GetMetric(ctx context.Context, name string) (metric Metric, err error) {
	rows, err := db.Connection.QueryContext(ctx,
		"SELECT m_name, m_value, m_type FROM metrics WHERE m_name = $1 LIMIT 1", name)
	if err != nil {
		return
	}

	var mN, mV, mT string
	var mValue interface{}
	if !rows.Next() {
		return metric, ErrMetricNotFound
	}
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
	return *m, nil
}

func (db *SQLStorage) BatchUpdate(ctx context.Context, metrics []Metric) (err error) {
	existedMetrics, err := db.GetAll(ctx)
	if err != nil {
		return
	}

	tx, err := db.Connection.BeginTx(ctx, nil)
	if err != nil {
		return
	}
	defer tx.Rollback()

	metricsUpdate := map[string]Metric{}
	for _, metric := range metrics {
		if metricUpdate, ok := metricsUpdate[metric.Name]; ok {
			err = metricUpdate.Update(metric.Value)
			if err != nil {
				return
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
				return
			}
			mN, mV, mT = existedMetric.GetMetricParamsString()
			if _, err = tx.ExecContext(ctx,
				"UPDATE metrics SET m_value = $2, m_type = $3 WHERE m_name = $1", mN, mV, mT); err != nil {
				return
			}
		} else {
			mN, mV, mT = metric.GetMetricParamsString()
			if _, err = tx.ExecContext(ctx,
				"INSERT INTO metrics(m_name, m_value, m_type) VALUES($1, $2, $3)", mN, mV, mT); err != nil {
				return
			}
		}
	}

	return tx.Commit()
}
