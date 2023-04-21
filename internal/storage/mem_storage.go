package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
)

type MemStorage struct {
	Metrics map[string]Metric
}

func (ms *MemStorage) AddMetric(_ context.Context, metric Metric) (err error) {
	if mInStorage, _ := ms.IsMetricInStorage(nil, metric); mInStorage {
		return fmt.Errorf("metric with name '%s' already present in Storage", metric.Name)
	}
	ms.Metrics[metric.Name] = metric
	return
}

func (ms *MemStorage) UpdateMetric(_ context.Context, metric Metric) (err error) {
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

func (ms *MemStorage) DeleteMetric(_ context.Context, metric Metric) (err error) {
	if mInStorage, _ := ms.IsMetricInStorage(nil, metric); !mInStorage {
		return fmt.Errorf("there is no metric with name '%s'", metric.Name)
	}
	delete(ms.Metrics, metric.Name)
	return
}

func (ms *MemStorage) IsMetricInStorage(_ context.Context, metric Metric) (bool, error) {
	_, isMetricExist := ms.Metrics[metric.Name]
	return isMetricExist, nil
}

// UpdateOrAddMetric Обновляет метрику, если она есть в коллекции, иначе добавляет ее.
func (ms *MemStorage) UpdateOrAddMetric(_ context.Context, metric Metric) (err error) {
	if mInStorage, _ := ms.IsMetricInStorage(nil, metric); mInStorage {
		err = ms.UpdateMetric(nil, metric)
	} else {
		err = ms.AddMetric(nil, metric)
	}
	return
}

func (ms *MemStorage) GetAll(_ context.Context) (map[string]Metric, error) {
	return ms.Metrics, nil
}

func (ms *MemStorage) GetMetric(_ context.Context, name string) (metric Metric, err error) {
	metric, ok := ms.Metrics[name]
	if !ok {
		return metric, ErrMetricNotFound
	}
	return
}

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

func (ms *MemStorage) BatchUpdate(_ context.Context, metrics []Metric) (err error) {
	for _, metric := range metrics {
		err = ms.UpdateOrAddMetric(nil, metric)
		if err != nil {
			return
		}
	}
	return
}

func NewMemStorage(metrics map[string]Metric) *MemStorage {
	return &MemStorage{Metrics: metrics}
}
