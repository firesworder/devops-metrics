package storage

import (
	"fmt"
)

type MemStorage struct {
	metrics map[string]Metric
}

func (ms *MemStorage) AddMetric(metric Metric) (err error) {
	if ms.IsMetricInStorage(metric) {
		return fmt.Errorf("metric with name '%s' already present in Storage", metric.Name)
	}
	ms.metrics[metric.Name] = metric
	return
}

func (ms *MemStorage) UpdateMetric(metric Metric) (err error) {
	metricToUpdate, ok := ms.metrics[metric.Name]
	if !ok {
		return fmt.Errorf("there is no metric with name '%s'", metric.Name)
	}
	err = metricToUpdate.Update(metric.Value)
	if err != nil {
		return err
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
