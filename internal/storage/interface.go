package storage

import "context"

type MetricRepository interface {
	AddMetric(context.Context, Metric) error
	UpdateMetric(context.Context, Metric) error
	DeleteMetric(context.Context, Metric) error

	IsMetricInStorage(context.Context, Metric) (bool, error)
	UpdateOrAddMetric(context.Context, Metric) error

	GetAll(context.Context) (map[string]Metric, error)
	GetMetric(context.Context, string) (Metric, error)
	BatchUpdate(context.Context, []Metric) error
}
