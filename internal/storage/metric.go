package storage

import (
	"fmt"
	"reflect"
)

type gauge float64
type counter int64

type Metric struct {
	Name  string
	Value interface{}
}

// todo: более гибко обрабатывать. Забрать парсинг из хандлера
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
