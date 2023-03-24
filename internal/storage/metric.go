package storage

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

// todo: переписать ошибки на статические

type gauge float64
type counter int64

var (
	ErrUnhandledValueType = errors.New("unhandled value type")
)

type Metric struct {
	Name  string
	Value interface{}
}

func NewMetric(name string, typeName string, rawValue interface{}) (*Metric, error) {
	var metricValue interface{}
	switch typeName {
	case "counter":
		switch castedValue := rawValue.(type) {
		case string:
			valueInt, err := strconv.ParseInt(castedValue, 10, 64)
			if err != nil {
				return nil, err
			}
			metricValue = counter(valueInt)
		case int64:
			metricValue = counter(castedValue)
		default:
			return nil, fmt.Errorf("cannot convert value '%T':'%v' to 'counter' type", rawValue, rawValue)
		}
	case "gauge":
		switch castedValue := rawValue.(type) {
		case string:
			valueFloat, err := strconv.ParseFloat(castedValue, 64)
			if err != nil {
				return nil, err
			}
			metricValue = gauge(valueFloat)
		case float64:
			metricValue = gauge(castedValue)
		default:
			return nil, fmt.Errorf("cannot convert value '%v' to 'gauge' type", rawValue)
		}
	default:
		return nil, fmt.Errorf("%w '%s'", ErrUnhandledValueType, typeName)
	}
	return &Metric{Name: name, Value: metricValue}, nil
}

func (m *Metric) Update(value interface{}) error {
	if reflect.TypeOf(m.Value) != reflect.TypeOf(value) {
		return fmt.Errorf("current(%T) and new(%T) value type mismatch",
			m.Value, value)
	}

	switch value := value.(type) {
	case gauge:
		m.Value = value
	case counter:
		m.Value = m.Value.(counter) + value
	}
	return nil
}
