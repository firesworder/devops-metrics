package storage

import "errors"

var (
	ErrMetricNotFound     = errors.New("metric was not found")
	ErrUnhandledValueType = errors.New("unhandled value type")
)
