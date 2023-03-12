package storage

import (
	"fmt"
	"reflect"
)

var MetricStorage *MemStorage

func init() {
	MetricStorage = NewMemStorage(map[string]Metric{})
}

type gauge float64
type counter int64

type Metric struct {
	name  string
	value interface{}
}

func NewMetric(name string, typeName string, rawValue interface{}) (*Metric, error) {
	var value interface{}
	switch typeName {
	case "counter":
		valueInt, ok := rawValue.(int64)
		if !ok {
			return nil, fmt.Errorf("cannot convert value '%v' to 'counter' type", rawValue)
		}
		value = counter(valueInt)
	case "gauge":
		valueFloat, ok := rawValue.(float64)
		if !ok {
			return nil, fmt.Errorf("cannot convert value '%v' to 'gauge' type", rawValue)
		}
		value = gauge(valueFloat)
	default:
		return nil, fmt.Errorf("unhandled value type '%s'", typeName)
	}
	return &Metric{name: name, value: value}, nil
}

type MetricRepository interface {
	AddMetric(Metric) error
	UpdateMetric(Metric) error
	DeleteMetric(Metric) error

	IsMetricInStorage(Metric) bool
	UpdateOrAddMetric(metric Metric) error

	GetAll() map[string]Metric
	GetMetric(string) (Metric, bool)
}

type MemStorage struct {
	metrics map[string]Metric
}

func (ms *MemStorage) AddMetric(metric Metric) (err error) {
	if ms.IsMetricInStorage(metric) {
		return fmt.Errorf("metric with name '%s' already present in Storage", metric.name)
	}

	switch metric.value.(type) {
	case counter, gauge:
		ms.metrics[metric.name] = metric
	default:
		return fmt.Errorf("unhandled value type '%T'", metric.value)
	}
	return
}

func (ms *MemStorage) UpdateMetric(metric Metric) (err error) {
	metricToUpdate, ok := ms.metrics[metric.name]
	if !ok {
		return fmt.Errorf("there is no metric with name '%s'", metric.name)
	}

	if reflect.TypeOf(metricToUpdate.value) != reflect.TypeOf(metric.value) {
		return fmt.Errorf("updated(%T) and new(%T) value type mismatch",
			metricToUpdate.value, metric.value)
	}

	switch value := metric.value.(type) {
	case gauge:
		metricToUpdate.value = value
	case counter:
		metricToUpdate.value = metricToUpdate.value.(counter) + value
	}
	ms.metrics[metric.name] = metricToUpdate
	return
}

func (ms *MemStorage) DeleteMetric(metric Metric) (err error) {
	if !ms.IsMetricInStorage(metric) {
		return fmt.Errorf("there is no metric with name '%s'", metric.name)
	}
	delete(ms.metrics, metric.name)
	return
}

func (ms *MemStorage) IsMetricInStorage(metric Metric) bool {
	_, isMetricExist := ms.metrics[metric.name]
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
	return ms.metrics
}

func (ms *MemStorage) GetMetric(name string) (metric Metric, ok bool) {
	metric, ok = ms.metrics[name]
	return
}

func NewMemStorage(metrics map[string]Metric) *MemStorage {
	return &MemStorage{metrics: metrics}
}
