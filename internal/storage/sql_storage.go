package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/firesworder/devopsmetrics/internal"
)

// SQLStorage реализует хранение и доступ к метрикам в SQL(Postgresql) БД.
// Доступно свойство Connection, для прямого доступа к БД(легаси, изначально предназначалось для хандлера Ping).
// BUG(firesworder): убрать прямой доступ к БД, если нужна команда Ping - реализовать через интерфейс MetricRepository.
type SQLStorage struct {
	Connection *sql.DB
}

// NewSQLStorage конструктор для SQLStorage.
// Открывает подключение к БД по DSN и создает таблицы для метрик, если они не существуют.
func NewSQLStorage(DSN string) (*SQLStorage, error) {
	// Этот метод вызывается при инициализации сервера, поэтому использую общий контекст
	ctx := context.Background()

	db := SQLStorage{}
	err := db.openDBConnection(DSN)
	if err != nil {
		return nil, err
	}
	err = db.createTableIfNotExist(ctx)
	if err != nil {
		return nil, err
	}
	return &db, nil
}

// openDBConnection создает подключение к бд.
func (db *SQLStorage) openDBConnection(DSN string) error {
	var err error
	db.Connection, err = sql.Open("pgx", DSN)
	if err != nil {
		return err
	}
	return nil
}

// createTableIfNotExist создает таблицу(metrics) для хранения метрик, если она еще не создана.
func (db *SQLStorage) createTableIfNotExist(ctx context.Context) (err error) {
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

// MetricRepository реализация.

// AddMetric добавляет метрику.
func (db *SQLStorage) AddMetric(ctx context.Context, metric Metric) (err error) {
	mN, mV, mT := metric.GetMetricParamsString()
	_, err = db.Connection.ExecContext(ctx,
		"INSERT INTO metrics(m_name, m_value, m_type) VALUES($1, $2, $3)", mN, mV, mT)
	if err != nil {
		return
	}
	return
}

// UpdateMetric обновляет значение метрики.
// Сначала метрика(переданная в арг-ах функции) находится в БД, затем обновляется в коде и обновление записывается в БД.
// BUG(firesworder): возвращается кастомная ошибка вместо ErrMetricNotFound.
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

// DeleteMetric удаляет метрику.
// Если метрика не найдена - возвращает ошибку ErrMetricNotFound.
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

// IsMetricInStorage возвращает true, если метрика с таким названием есть в таблице, иначе false.
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

// UpdateOrAddMetric добавляет или обновляет(если есть в БД) метрику.
// Обертка над AddMetric и UpdateMetric.
// Сначала проверяется наличие записи метрики в БД, потом происходит либо добавление либо обновление.
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

// GetAll возвращает все метрики в таблице.
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

// GetMetric возвращает метрику с названием `name` из таблицы.
// Если метрика не найдена - возвращает ошибку ErrMetricNotFound.
func (db *SQLStorage) GetMetric(ctx context.Context, name string) (metric Metric, err error) {
	var mN, mV, mT string
	var mValue interface{}
	err = db.Connection.QueryRowContext(ctx,
		"SELECT m_name, m_value, m_type FROM metrics WHERE m_name = $1 LIMIT 1", name).Scan(&mN, &mV, &mT)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return metric, ErrMetricNotFound
		}
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

	return *m, nil
}

// BatchUpdate обновляет метрики в таблице батчем metrics.
// Обрабатан кейс нескольких обновлений одной и той же метрики.
//
// Сначала запрашиваются все метрики, затем формируется мап "имя метрики" => <суммарное изменение метрики>
// и далее метрики либо добавляются либо обновляются(если были в БД на момент обновления).
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
