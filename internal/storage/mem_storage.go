package storage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/firesworder/devopsmetrics/internal"
)

// MemStorage реализует хранение и доступ к метрикам в памяти.
// Доступ и хранение обеспечиваются посредством мапа Metrics, а также методами, по интерфейсу MetricRepository.
type MemStorage struct {
	Metrics map[string]Metric
}

// AddMetric добавляет метрику.
// Если ключ с названием метрики уже в мапе - возвращает ошибку.
func (ms *MemStorage) AddMetric(ctx context.Context, metric Metric) (err error) {
	if mInStorage, _ := ms.IsMetricInStorage(ctx, metric); mInStorage {
		return fmt.Errorf("metric with name '%s' already present in Storage", metric.Name)
	}
	ms.Metrics[metric.Name] = metric
	return
}

// UpdateMetric обновляет метрику.
// Если ключ с названием метрики не найден в мапе - возвращает ошибку.
func (ms *MemStorage) UpdateMetric(ctx context.Context, metric Metric) (err error) {
	metricToUpdate, ok := ms.Metrics[metric.Name]
	if !ok {
		return fmt.Errorf("there is no metric with name '%s'", metric.Name)
	}
	err = metricToUpdate.Update(metric.Value)
	if err != nil {
		return err
	}
	ms.Metrics[metric.Name] = metricToUpdate
	return
}

// DeleteMetric удаляет метрику из мапа.
// Если ключ с названием метрики не найден в мапе - возвращает ошибку.
func (ms *MemStorage) DeleteMetric(ctx context.Context, metric Metric) (err error) {
	if mInStorage, _ := ms.IsMetricInStorage(ctx, metric); !mInStorage {
		return fmt.Errorf("there is no metric with name '%s'", metric.Name)
	}
	delete(ms.Metrics, metric.Name)
	return
}

// IsMetricInStorage возвращает true если метрика с таким названием присутствует в мапе, иначе false.
// Ошибка не генерируется.
func (ms *MemStorage) IsMetricInStorage(ctx context.Context, metric Metric) (bool, error) {
	_, isMetricExist := ms.Metrics[metric.Name]
	return isMetricExist, nil
}

// UpdateOrAddMetric Обновляет метрику, если она есть в коллекции, иначе добавляет ее.
// Ошибка не генерируется.
func (ms *MemStorage) UpdateOrAddMetric(ctx context.Context, metric Metric) (err error) {
	if mInStorage, _ := ms.IsMetricInStorage(ctx, metric); mInStorage {
		_ = ms.UpdateMetric(ctx, metric)
	} else {
		_ = ms.AddMetric(ctx, metric)
	}
	return
}

// GetAll возвращет все метрики.
// Ошибка не генерируется.
func (ms *MemStorage) GetAll(ctx context.Context) (map[string]Metric, error) {
	return ms.Metrics, nil
}

// GetMetric возвращает метрику из репозитория по названию `name`.
// Если метрика не найдена - возвращает ошибку ErrMetricNotFound.
func (ms *MemStorage) GetMetric(ctx context.Context, name string) (metric Metric, err error) {
	metric, ok := ms.Metrics[name]
	if !ok {
		return metric, ErrMetricNotFound
	}
	return
}

// MarshalJSON реализация интерфейса json.Marshaler.
// Используется для сохранения состояния репозитория в файл(filestore.FileStore).
func (ms *MemStorage) MarshalJSON() ([]byte, error) {
	type extendedMetric struct {
		Metric
		ValueType string
	}

	type MemStorageExt struct {
		Metrics map[string]extendedMetric
	}

	mse := MemStorageExt{Metrics: map[string]extendedMetric{}}
	var valueType string
	var extM extendedMetric
	for _, m := range ms.Metrics {
		switch m.Value.(type) {
		case counter:
			valueType = internal.CounterTypeName
		case gauge:
			valueType = internal.GaugeTypeName
		default:
			return nil, ErrUnhandledValueType
		}
		extM = extendedMetric{Metric: m, ValueType: valueType}
		mse.Metrics[extM.Name] = extM
	}

	return json.Marshal(mse)
}

// UnmarshalJSON реализация интерфейса json.Unmarshaler.
// Используется для загрузки состояния репозитория из файла(filestore.FileStore).
func (ms *MemStorage) UnmarshalJSON(data []byte) error {
	type extendedMetric struct {
		Metric
		ValueType string
	}

	type MemStorageExt struct {
		Metrics map[string]extendedMetric
	}

	mse := MemStorageExt{Metrics: map[string]extendedMetric{}}

	err := json.Unmarshal(data, &mse)
	if err != nil {
		return err
	}

	var metric Metric
	for _, extM := range mse.Metrics {
		metric = Metric{Name: extM.Name}
		switch extM.ValueType {
		case internal.CounterTypeName:
			metric.Value = counter(extM.Value.(float64))
		case internal.GaugeTypeName:
			metric.Value = gauge(extM.Value.(float64))
		default:
			return ErrUnhandledValueType
		}
		ms.Metrics[metric.Name] = metric
	}

	return nil
}

// BatchUpdate обновляет метрики в репозитории батчем.
// Ошибка не генерируется.
func (ms *MemStorage) BatchUpdate(ctx context.Context, metrics []Metric) (err error) {
	for _, metric := range metrics {
		err = ms.UpdateOrAddMetric(ctx, metric)
		if err != nil {
			return
		}
	}
	return
}

// NewMemStorage конструктор.
// Deprecated: был актуален, когда свойство Metrics было приватным.
// Можно создавать напрямую объект MemStorage.
func NewMemStorage(metrics map[string]Metric) *MemStorage {
	return &MemStorage{Metrics: metrics}
}
