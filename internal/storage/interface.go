package storage

import "context"

// MetricRepository интерфейс взаимодействия с репозиторием(коллекцией) метрик.
type MetricRepository interface {
	// AddMetric добавляет метрику.
	AddMetric(context.Context, Metric) error
	// UpdateMetric обновляет значение метрики в репозитории.
	UpdateMetric(context.Context, Metric) error
	// DeleteMetric удаляет метрику.
	DeleteMetric(context.Context, Metric) error

	// IsMetricInStorage проверяет наличие метрики.
	IsMetricInStorage(context.Context, Metric) (bool, error)
	// UpdateOrAddMetric обертка над методами AddMetric и UpdateMetric.
	// Если метрика в репозитории - обновляет ее, иначе добавляет.
	UpdateOrAddMetric(context.Context, Metric) error

	// GetAll возвращает все метрики.
	GetAll(context.Context) (map[string]Metric, error)
	// GetMetric возвращает метрику по названию.
	GetMetric(context.Context, string) (Metric, error)
	// BatchUpdate обновляет репозиторий элементами слайса метрик.
	BatchUpdate(context.Context, []Metric) error
}
