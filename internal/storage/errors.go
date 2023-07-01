package storage

import "errors"

var (
	// ErrMetricNotFound ошибка "метрика с данным названием не найдена"
	ErrMetricNotFound = errors.New("metric was not found")
	// ErrUnhandledValueType ошибка "указан нереализованный тип метрик"
	ErrUnhandledValueType = errors.New("unhandled value type")
)
