package storage

import (
	"errors"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/message"
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
			return nil, fmt.Errorf("cannot convert value '%T':'%v' to 'gauge' type", rawValue, rawValue)
		}
	default:
		return nil, fmt.Errorf("%w '%s'", ErrUnhandledValueType, typeName)
	}
	return &Metric{Name: name, Value: metricValue}, nil
}

// todo: описать тесты
func NewMetricFromMessage(metrics *message.Metrics) (newMetric *Metric, err error) {
	switch metrics.MType {
	case "counter":
		if metrics.Delta == nil {
			return nil, fmt.Errorf("param 'delta' cannot be nil for type 'counter'")
		}
		newMetric, err = NewMetric(metrics.ID, metrics.MType, *metrics.Delta)
	case "gauge":
		if metrics.Value == nil {
			return nil, fmt.Errorf("param 'value' cannot be nil for type 'gauge'")
		}
		newMetric, err = NewMetric(metrics.ID, metrics.MType, *metrics.Value)
	default:
		return nil, fmt.Errorf("%w '%s'", ErrUnhandledValueType, metrics.MType)
	}
	return
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

func (m *Metric) GetMessageMetric() (messageMetric message.Metrics) {
	messageMetric.ID = m.Name
	switch value := m.Value.(type) {
	case gauge:
		messageMetric.MType = "gauge"
		mValue := float64(value)
		messageMetric.Value = &mValue
	case counter:
		messageMetric.MType = "counter"
		mDelta := int64(value)
		messageMetric.Delta = &mDelta
	}
	return
}
