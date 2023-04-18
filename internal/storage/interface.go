package storage

type MetricRepository interface {
	AddMetric(Metric) error
	UpdateMetric(Metric) error
	DeleteMetric(Metric) error

	IsMetricInStorage(Metric) bool
	UpdateOrAddMetric(metric Metric) error

	GetAll() map[string]Metric
	GetMetric(string) (Metric, bool)
	BatchUpdate([]Metric) error
}
