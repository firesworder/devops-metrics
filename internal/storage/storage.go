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
	Name  string
	Value interface{}
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
	return &Metric{Name: name, Value: value}, nil
}

type MemStorage struct {
	metrics map[string]Metric
}

func (ms *MemStorage) AddMetric(metric Metric) (err error) {
	if ms.IsMetricInStorage(metric) {
		return fmt.Errorf("metric with name '%s' already present in Storage", metric.Name)
	}

	switch metric.Value.(type) {
	case counter, gauge:
		ms.metrics[metric.Name] = metric
	default:
		return fmt.Errorf("unhandled value type '%T'", metric.Value)
	}
	return
}

func (ms *MemStorage) UpdateMetric(metric Metric) (err error) {
	metricToUpdate, ok := ms.metrics[metric.Name]
	if !ok {
		return fmt.Errorf("there is no metric with name '%s'", metric.Name)
	}

	if reflect.TypeOf(metricToUpdate.Value) != reflect.TypeOf(metric.Value) {
		return fmt.Errorf("updated(%T) and new(%T) value type mismatch",
			metricToUpdate.Value, metric.Value)
	}

	switch value := metric.Value.(type) {
	case gauge:
		metricToUpdate.Value = value
	case counter:
		metricToUpdate.Value = metricToUpdate.Value.(counter) + value
	}
	ms.metrics[metric.Name] = metricToUpdate
	return
}

func (ms *MemStorage) DeleteMetric(metric Metric) (err error) {
	if !ms.IsMetricInStorage(metric) {
		return fmt.Errorf("there is no metric with name '%s'", metric.Name)
	}
	delete(ms.metrics, metric.Name)
	return
}

func (ms *MemStorage) IsMetricInStorage(metric Metric) bool {
	_, isMetricExist := ms.metrics[metric.Name]
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
