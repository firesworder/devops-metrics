package storage

import (
	"fmt"
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

func NewMetric(name string, typeName string, rawValue interface{}) *Metric {
	var value interface{}
	switch typeName {
	case "counter":
		value = counter(rawValue.(int64))
	case "gauge":
		value = gauge(rawValue.(float64))
	}
	return &Metric{name: name, value: value}
}

type MetricRepository interface {
	AddMetric(Metric)
	UpdateMetric(Metric)
	DeleteMetric(Metric)
}

type MemStorage struct {
	metrics map[string]Metric
}

func (ms *MemStorage) AddMetric(metric Metric) {
	if !ms.IsMetricInStorage(metric) {
		ms.metrics[metric.name] = metric
	}
}

func (ms *MemStorage) UpdateMetric(metric Metric) {
	metricToUpdate, ok := ms.metrics[metric.name]
	if !ok {
		return
	}

	switch value := metric.value.(type) {
	case gauge:
		metricToUpdate.value = value
	case counter:
		metricToUpdate.value = metricToUpdate.value.(counter) + value
	}
	ms.metrics[metric.name] = metricToUpdate
}

func (ms *MemStorage) DeleteMetric(metric Metric) {
	if ms.IsMetricInStorage(metric) {
		delete(ms.metrics, metric.name)
	}
}

func (ms *MemStorage) IsMetricInStorage(metric Metric) bool {
	_, isMetricExist := ms.metrics[metric.name]
	return isMetricExist
}

// UpdateOrAddMetric Обновляет метрику, если она есть в коллекции, иначе добавляет ее.
func (ms *MemStorage) UpdateOrAddMetric(metric Metric) {
	if ms.IsMetricInStorage(metric) {
		ms.UpdateMetric(metric)
	} else {
		ms.AddMetric(metric)
	}
}

func (ms *MemStorage) getAll() map[string]Metric {
	return ms.metrics
}

func (ms *MemStorage) getMetric(name string) (metric Metric, ok bool) {
	metric, ok = ms.metrics[name]
	return
}

func NewMemStorage(metrics map[string]Metric) *MemStorage {
	return &MemStorage{metrics: metrics}
}

// todo: удалить по завершению. Можно использовать для описания тестов
func Playground() {
	memStorage := NewMemStorage(map[string]Metric{})
	demoMetric := Metric{name: "demo", value: gauge(1.12)}
	memStorage.AddMetric(demoMetric)
	fmt.Println(memStorage)

	demoMetric.value = gauge(1.45)
	memStorage.UpdateMetric(demoMetric)
	fmt.Println(memStorage)

	metricToDelete := Metric{name: "toDelete", value: counter(10)}
	memStorage.AddMetric(metricToDelete)
	fmt.Println(memStorage)
	memStorage.DeleteMetric(metricToDelete)
	fmt.Println(memStorage)
}
