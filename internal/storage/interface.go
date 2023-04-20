package storage

import "context"

type MetricRepository interface {
	AddMetric(context.Context, Metric) error
	UpdateMetric(context.Context, Metric) error
	DeleteMetric(context.Context, Metric) error

	IsMetricInStorage(context.Context, Metric) bool
	UpdateOrAddMetric(context.Context, Metric) error

	GetAll(context.Context) map[string]Metric
	GetMetric(context.Context, string) (Metric, bool)
	BatchUpdate(context.Context, []Metric) error
}
