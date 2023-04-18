package storage

import (
	"encoding/json"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
)

type MemStorage struct {
	Metrics map[string]Metric
}

func (ms *MemStorage) AddMetric(metric Metric) (err error) {
	if ms.IsMetricInStorage(metric) {
		return fmt.Errorf("metric with name '%s' already present in Storage", metric.Name)
	}
	ms.Metrics[metric.Name] = metric
	return
}

func (ms *MemStorage) UpdateMetric(metric Metric) (err error) {
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

func (ms *MemStorage) DeleteMetric(metric Metric) (err error) {
	if !ms.IsMetricInStorage(metric) {
		return fmt.Errorf("there is no metric with name '%s'", metric.Name)
	}
	delete(ms.Metrics, metric.Name)
	return
}

func (ms *MemStorage) IsMetricInStorage(metric Metric) bool {
	_, isMetricExist := ms.Metrics[metric.Name]
	return isMetricExist
}

// UpdateOrAddMetric Обновляет метрику, если она есть в коллекции, иначе добавляет ее.
func (ms *MemStorage) UpdateOrAddMetric(metric Metric) (err error) {
	if ms.IsMetricInStorage(metric) {
		err = ms.UpdateMetric(metric)
	} else {
		err = ms.AddMetric(metric)
	}
	return
}

func (ms *MemStorage) GetAll() map[string]Metric {
	return ms.Metrics
}

func (ms *MemStorage) GetMetric(name string) (metric Metric, ok bool) {
	metric, ok = ms.Metrics[name]
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

func (ms *MemStorage) BatchUpdate(metrics map[string]Metric) (err error) {
	for _, metric := range metrics {
		err = ms.UpdateOrAddMetric(metric)
		if err != nil {
			return
		}
	}
	return
}

func NewMemStorage(metrics map[string]Metric) *MemStorage {
	return &MemStorage{Metrics: metrics}
}
