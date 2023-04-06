package storage

import (
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

func TestMetric_Update(t *testing.T) {
	tests := []struct {
		name          string
		updatedMetric Metric
		newValue      interface{}
		wantMetric    Metric
		wantError     error
	}{
		{
			name:          "Test 1. Correct update, type counter",
			updatedMetric: Metric{Name: "metric1", Value: counter(10)},
			newValue:      counter(15),
			wantMetric:    Metric{Name: "metric1", Value: counter(25)},
			wantError:     nil,
		},
		{
			name:          "Test 2. Correct update, type gauge",
			updatedMetric: Metric{Name: "metric1", Value: gauge(12.3)},
			newValue:      gauge(15.5),
			wantMetric:    Metric{Name: "metric1", Value: gauge(15.5)},
			wantError:     nil,
		},
		{
			name:          "Test 3. Type of updated metric and new value differ",
			updatedMetric: Metric{Name: "metric1", Value: counter(10)},
			newValue:      gauge(15.5),
			wantMetric:    Metric{Name: "metric1", Value: counter(10)},
			wantError: fmt.Errorf("current(%T) and new(%T) value type mismatch",
				counter(10), gauge(15.5)),
		},
		{
			name:          "Test 4. Metric with unhandled value type, incl nil",
			updatedMetric: Metric{Name: "metric1", Value: nil},
			newValue:      gauge(15.5),
			wantMetric:    Metric{Name: "metric1", Value: nil},
			wantError: fmt.Errorf("current(%T) and new(%T) value type mismatch",
				nil, gauge(15.5)),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.updatedMetric.Update(tt.newValue)
			assert.Equal(t, tt.wantMetric, tt.updatedMetric)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

func TestNewMetric(t *testing.T) {
	type args struct {
		name     string
		typeName string
		rawValue interface{}
	}
	tests := []struct {
		name      string
		args      args
		want      *Metric
		wantError error
	}{
		// Counter
		{
			name:      "Test correct counter #1. Correct int64 value.",
			args:      args{name: "testMetric11", typeName: internal.CounterTypeName, rawValue: int64(10)},
			want:      &Metric{Name: "testMetric11", Value: counter(10)},
			wantError: nil,
		},
		{
			name:      "Test correct counter #2. Correct string(int) value.",
			args:      args{name: "testMetric11", typeName: internal.CounterTypeName, rawValue: "10"},
			want:      &Metric{Name: "testMetric11", Value: counter(10)},
			wantError: nil,
		},

		{
			name:      "Test incorrect counter #1. Incorrect number value.",
			args:      args{name: "testMetric12", typeName: internal.CounterTypeName, rawValue: 11.3},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value 'float64':'11.3' to 'counter' type"),
		},
		{
			name:      "Test incorrect counter #2. Incorrect string(NAN) value.",
			args:      args{name: "testMetric13", typeName: internal.CounterTypeName, rawValue: "str"},
			want:      nil,
			wantError: &strconv.NumError{Num: "str", Func: "ParseInt", Err: fmt.Errorf("invalid syntax")},
		},
		{
			name:      "Test incorrect counter #3. Incorrect nil value type.",
			args:      args{name: "testMetric1", typeName: internal.CounterTypeName, rawValue: nil},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '<nil>':'<nil>' to 'counter' type"),
		},
		{
			name:      "Test incorrect counter #4. Incorrect int(not int64!) value.",
			args:      args{name: "testMetric11", typeName: internal.CounterTypeName, rawValue: int(10)},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value 'int':'10' to 'counter' type"),
		},
		{
			name:      "Test incorrect counter #5. Incorrect string(not int) value.",
			args:      args{name: "testMetric11", typeName: internal.CounterTypeName, rawValue: "10.2"},
			want:      nil,
			wantError: &strconv.NumError{Num: "10.2", Func: "ParseInt", Err: fmt.Errorf("invalid syntax")},
		},

		// Gauge
		{
			name:      "Test correct gauge #1. Correct float64 value.",
			args:      args{name: "testMetric2", typeName: internal.GaugeTypeName, rawValue: 11.2},
			want:      &Metric{Name: "testMetric2", Value: gauge(11.2)},
			wantError: nil,
		},
		{
			name:      "Test correct gauge #2. Correct(float) string value.",
			args:      args{name: "testMetric11", typeName: internal.GaugeTypeName, rawValue: "11.2"},
			want:      &Metric{Name: "testMetric11", Value: gauge(11.2)},
			wantError: nil,
		},
		{
			name:      "Test correct gauge #3. Int string value.",
			args:      args{name: "testMetric11", typeName: internal.GaugeTypeName, rawValue: "10"},
			want:      &Metric{Name: "testMetric11", Value: gauge(10)},
			wantError: nil,
		},

		{
			name:      "Test incorrect gauge #1. Incorrect number value.",
			args:      args{name: "testMetric2", typeName: internal.GaugeTypeName, rawValue: 10},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value 'int':'10' to 'gauge' type"),
		},
		{
			name:      "Test incorrect gauge #2. Incorrect NAN value.",
			args:      args{name: "testMetric2", typeName: internal.GaugeTypeName, rawValue: "str"},
			want:      nil,
			wantError: &strconv.NumError{Num: "str", Func: "ParseFloat", Err: fmt.Errorf("invalid syntax")},
		},
		{
			name:      "Test incorrect gauge #3. Incorrect nil value type.",
			args:      args{name: "testMetric2", typeName: internal.GaugeTypeName, rawValue: nil},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '<nil>':'<nil>' to 'gauge' type"),
		},
		{
			name:      "Test incorrect gauge #4. Incorrect type float32(instead of float64) value.",
			args:      args{name: "testMetric2", typeName: internal.GaugeTypeName, rawValue: float32(11.2)},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value 'float32':'11.2' to 'gauge' type"),
		},

		// others
		{
			name:      "Test others #1. Unknown value type.",
			args:      args{name: "testMetric2", typeName: "int", rawValue: 100},
			want:      nil,
			wantError: fmt.Errorf("%w '%s'", ErrUnhandledValueType, "int"),
		},
		{
			name:      "Test others #2. Empty name.",
			args:      args{name: "", typeName: internal.CounterTypeName, rawValue: int64(100)},
			want:      &Metric{Name: "", Value: counter(100)},
			wantError: nil,
		},
		{
			name:      "Test others #3. Empty type.",
			args:      args{name: "metric1", typeName: "", rawValue: int64(100)},
			want:      nil,
			wantError: fmt.Errorf("%w '%s'", ErrUnhandledValueType, ""),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMetric, err := NewMetric(tt.args.name, tt.args.typeName, tt.args.rawValue)
			assert.Equal(t, tt.want, gotMetric)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

func TestNewMetricFromMessage(t *testing.T) {
	float64Val, int64Val := 12.133, int64(10)
	tests := []struct {
		name       string
		message    *message.Metrics
		wantMetric *Metric
		wantErr    error
	}{
		{
			name:       "Test correct counter #1.",
			message:    &message.Metrics{ID: "PollCount", MType: internal.CounterTypeName, Delta: &int64Val},
			wantMetric: &Metric{Name: "PollCount", Value: counter(int64Val)},
			wantErr:    nil,
		},
		{
			name:       "Test correct gauge #1.",
			message:    &message.Metrics{ID: "RandomValue", MType: internal.GaugeTypeName, Value: &float64Val},
			wantMetric: &Metric{Name: "RandomValue", Value: gauge(float64Val)},
			wantErr:    nil,
		},
		{
			name:       "Test correct others #1. Empty name",
			message:    &message.Metrics{ID: "", MType: internal.CounterTypeName, Delta: &int64Val},
			wantMetric: &Metric{Name: "", Value: counter(int64Val)},
			wantErr:    nil,
		},

		{
			name:       "Test incorrect counter #1. No value params.",
			message:    &message.Metrics{ID: "PollCount", MType: internal.CounterTypeName},
			wantMetric: nil,
			wantErr:    fmt.Errorf("param 'delta' cannot be nil for type 'counter'"),
		},
		{
			name:       "Test incorrect counter #2. No 'delta' param, but 'value'.",
			message:    &message.Metrics{ID: "PollCount", MType: internal.CounterTypeName, Value: &float64Val},
			wantMetric: nil,
			wantErr:    fmt.Errorf("param 'delta' cannot be nil for type 'counter'"),
		},
		{
			name:       "Test incorrect gauge #1. No value params.",
			message:    &message.Metrics{ID: "RandomValue", MType: internal.GaugeTypeName},
			wantMetric: nil,
			wantErr:    fmt.Errorf("param 'value' cannot be nil for type 'gauge'"),
		},
		{
			name:       "Test incorrect gauge #2. No 'value' param.",
			message:    &message.Metrics{ID: "RandomValue", MType: internal.GaugeTypeName, Delta: &int64Val},
			wantMetric: nil,
			wantErr:    fmt.Errorf("param 'value' cannot be nil for type 'gauge'"),
		},
		{
			name:       "Test incorrect others #1. Unknown metric value type.",
			message:    &message.Metrics{ID: "RandomValue", MType: "sometype", Delta: &int64Val, Value: &float64Val},
			wantMetric: nil,
			wantErr:    fmt.Errorf("%w '%s'", ErrUnhandledValueType, "sometype"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric, err := NewMetricFromMessage(tt.message)
			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantMetric, metric)
		})
	}
}

func TestMetric_GetMessageMetric(t *testing.T) {
	int64Val, float64Val := int64(10), float64(12.133)
	tests := []struct {
		name              string
		metric            Metric
		wantMessageMetric message.Metrics
	}{
		{
			name:              "Test correct counter #1.",
			metric:            Metric{Name: "PollCount", Value: counter(int64Val)},
			wantMessageMetric: message.Metrics{ID: "PollCount", MType: internal.CounterTypeName, Delta: &int64Val},
		},
		{
			name:              "Test correct gauge #1.",
			metric:            Metric{Name: "RandomValue", Value: gauge(float64Val)},
			wantMessageMetric: message.Metrics{ID: "RandomValue", MType: internal.GaugeTypeName, Value: &float64Val},
		},

		{
			name:              "Test incorrect #1. Empty metric.",
			metric:            Metric{},
			wantMessageMetric: message.Metrics{},
		},
		{
			name:              "Test incorrect #2. Metric only with name.",
			metric:            Metric{Name: "Metric"},
			wantMessageMetric: message.Metrics{ID: "Metric"},
		},
		{
			name:              "Test incorrect #3. Metric with unknown type.",
			metric:            Metric{Name: "RandomValue", Value: int8(11)},
			wantMessageMetric: message.Metrics{ID: "RandomValue"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMessage := tt.metric.GetMessageMetric()
			assert.Equal(t, tt.wantMessageMetric, gotMessage)
		})
	}
}

func TestMetric_GetValueString(t *testing.T) {
	tests := []struct {
		name   string
		metric Metric
		want   string
	}{
		{
			name:   "Test correct counter.",
			metric: Metric{Name: "PollCount", Value: counter(10)},
			want:   "10",
		},
		{
			name:   "Test correct gauge.",
			metric: Metric{Name: "Alloc", Value: gauge(22.2)},
			want:   "22.200",
		},

		{
			name:   "Test incorrect. Unknown type.",
			metric: Metric{Name: "UnknownMetric", Value: int8(11)},
			want:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.metric.GetValueString())
		})
	}
}
