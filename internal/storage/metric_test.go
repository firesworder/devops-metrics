package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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
		{
			name:      "Test 1. Counter type metric, with correct value.",
			args:      args{name: "testMetric11", typeName: "counter", rawValue: int64(10)},
			want:      &Metric{Name: "testMetric11", Value: counter(10)},
			wantError: nil,
		},
		{
			name:      "Test 2. Counter type metric, with incorrect number value.",
			args:      args{name: "testMetric12", typeName: "counter", rawValue: 11.3},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '%v' to 'counter' type", 11.3),
		},
		{
			name:      "Test 3. Counter type metric, with incorrect NAN value.",
			args:      args{name: "testMetric13", typeName: "counter", rawValue: "str"},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '%v' to 'counter' type", "str"),
		},
		{
			name:      "Test 4. Gauge type metric, with correct value.",
			args:      args{name: "testMetric2", typeName: "gauge", rawValue: 11.2},
			want:      &Metric{Name: "testMetric2", Value: gauge(11.2)},
			wantError: nil,
		},
		{
			name:      "Test 5. Gauge type metric, with incorrect number value.",
			args:      args{name: "testMetric2", typeName: "gauge", rawValue: 10},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '%v' to 'gauge' type", 10),
		},
		{
			name:      "Test 6. Gauge type metric, with incorrect NAN value.",
			args:      args{name: "testMetric2", typeName: "gauge", rawValue: "str"},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '%v' to 'gauge' type", "str"),
		},
		{
			name:      "Test 7. Counter type metric, nil value type.",
			args:      args{name: "testMetric1", typeName: "counter", rawValue: nil},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '%v' to 'counter' type", nil),
		},
		{
			name:      "Test 8. Gauge type metric, nil value type.",
			args:      args{name: "testMetric2", typeName: "gauge", rawValue: nil},
			want:      nil,
			wantError: fmt.Errorf("cannot convert value '%v' to 'gauge' type", nil),
		},
		{
			name:      "Test 9. Unknown value type.",
			args:      args{name: "testMetric2", typeName: "int", rawValue: 100},
			want:      nil,
			wantError: fmt.Errorf("unhandled value type '%s'", "int"),
		},
		{
			name:      "Test 10. Empty name.",
			args:      args{name: "", typeName: "counter", rawValue: int64(100)},
			want:      &Metric{Name: "", Value: counter(100)},
			wantError: nil,
		},
		{
			name:      "Test 11. Empty type.",
			args:      args{name: "metric1", typeName: "", rawValue: int64(100)},
			want:      nil,
			wantError: fmt.Errorf("unhandled value type '%s'", ""),
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
