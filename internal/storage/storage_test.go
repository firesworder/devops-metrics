package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var testMetric11, testMetric12, testMetric13 Metric
var testMetric4, testMetric5, testMetric6, testMetric7 Metric

func init() {
	testMetric11 = Metric{Name: "testMetric1", Value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	testMetric12 = Metric{Name: "testMetric1", Value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	testMetric13 = Metric{Name: "testMetric1", Value: gauge(22.2)}

	testMetric4 = Metric{Name: "testMetric4", Value: gauge(2.27)}
	testMetric5 = Metric{Name: "testMetric5", Value: 0}
	testMetric6 = Metric{}
	testMetric7 = Metric{Name: "testMetric7", Value: counter(27)}
}

// todo: почистить

func TestMemStorage_AddMetric(t *testing.T) {
	tests := []struct {
		name        string
		metricToAdd Metric
		startState  map[string]Metric
		wantedState map[string]Metric
		wantError   error
	}{
		{
			name:        "Test 1. Add metric to empty storage state.",
			metricToAdd: testMetric11,
			startState:  map[string]Metric{},
			wantedState: map[string]Metric{testMetric11.Name: testMetric11},
			wantError:   nil,
		},
		{
			name:        "Test 2. Add metric to storage, but metric already present.",
			metricToAdd: testMetric12,
			startState:  map[string]Metric{testMetric11.Name: testMetric11},
			wantedState: map[string]Metric{testMetric11.Name: testMetric11},
			wantError:   fmt.Errorf("metric with name '%s' already present in Storage", testMetric12.Name),
		},
		{
			name:        "Test 3. Add metric to storage, but metric already present. Value type differ",
			metricToAdd: testMetric13,
			startState:  map[string]Metric{testMetric11.Name: testMetric11},
			wantedState: map[string]Metric{testMetric11.Name: testMetric11},
			wantError:   fmt.Errorf("metric with name '%s' already present in Storage", testMetric13.Name),
		},
		{
			name:        "Test 4. Add another metric to storage",
			metricToAdd: testMetric4,
			startState:  map[string]Metric{testMetric11.Name: testMetric11},
			wantedState: map[string]Metric{testMetric11.Name: testMetric11, testMetric4.Name: testMetric4},
			wantError:   nil,
		},
		{
			name:        "Test 5. Add metric with unhandled value type",
			metricToAdd: testMetric5,
			startState:  map[string]Metric{},
			wantedState: map[string]Metric{},
			wantError:   fmt.Errorf("unhandled value type '%T'", testMetric5.Value),
		},
		{
			name:        "Test 6. Add empty metric",
			metricToAdd: testMetric6,
			startState:  map[string]Metric{},
			wantedState: map[string]Metric{},
			wantError:   fmt.Errorf("unhandled value type '%T'", testMetric6.Value),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.startState,
			}
			err := ms.AddMetric(tt.metricToAdd)
			assert.Equal(t, tt.wantedState, ms.metrics)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

func TestMemStorage_DeleteMetric(t *testing.T) {
	tests := []struct {
		name           string
		metricToDelete Metric
		startState     map[string]Metric
		wantedState    map[string]Metric
		wantError      error
	}{
		{
			name:           "Test 1. Delete metric from state contains ONLY that metric.",
			metricToDelete: testMetric11,
			startState:     map[string]Metric{testMetric11.Name: testMetric11},
			wantedState:    map[string]Metric{},
			wantError:      nil,
		},
		{
			name:           "Test 2. Delete metric from state that contains that metric.",
			metricToDelete: testMetric11,
			startState: map[string]Metric{
				testMetric11.Name: testMetric11,
				testMetric4.Name:  testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric4.Name: testMetric4,
			},
			wantError: nil,
		},
		{
			name:           "Test 3. Delete metric from state that contains metrics, except that metric.",
			metricToDelete: testMetric11,
			startState: map[string]Metric{
				testMetric7.Name: testMetric7,
				testMetric4.Name: testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric7.Name: testMetric7,
				testMetric4.Name: testMetric4,
			},
			wantError: fmt.Errorf("there is no metric with name '%s'", testMetric11.Name),
		},
		{
			name:           "Test 4. Delete metric from state contains that metric, but value differ.",
			metricToDelete: testMetric11,
			startState: map[string]Metric{
				testMetric13.Name: testMetric13,
				testMetric4.Name:  testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric4.Name: testMetric4,
			},
			wantError: nil,
		},
		{
			name:           "Test 5. Delete metric from empty state.",
			metricToDelete: testMetric11,
			startState:     map[string]Metric{},
			wantedState:    map[string]Metric{},
			wantError:      fmt.Errorf("there is no metric with name '%s'", testMetric11.Name),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.startState,
			}
			err := ms.DeleteMetric(tt.metricToDelete)
			assert.Equal(t, tt.wantedState, ms.metrics)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

func TestMemStorage_IsMetricInStorage(t *testing.T) {
	tests := []struct {
		name          string
		metricToCheck Metric
		startState    map[string]Metric
		wantedResult  bool
	}{
		{
			name:          "Test 1. Searched metric present in state. State contains only that metric.",
			metricToCheck: testMetric11,
			startState:    map[string]Metric{testMetric11.Name: testMetric11},
			wantedResult:  true,
		},
		{
			name:          "Test 2. Searched metric present in state. Multiple metrics in state.",
			metricToCheck: testMetric11,
			startState:    map[string]Metric{testMetric11.Name: testMetric11, testMetric4.Name: testMetric4},
			wantedResult:  true,
		},
		{
			name:          "Test 3. Metric name present in state, but value differs",
			metricToCheck: testMetric11,
			startState:    map[string]Metric{testMetric13.Name: testMetric13, testMetric4.Name: testMetric4},
			wantedResult:  true,
		},
		{
			name:          "Test 4. Metric is not present in state.",
			metricToCheck: testMetric11,
			startState:    map[string]Metric{testMetric7.Name: testMetric7, testMetric4.Name: testMetric4},
			wantedResult:  false,
		},
		{
			name:          "Test 5. Empty state.",
			metricToCheck: testMetric11,
			startState:    map[string]Metric{},
			wantedResult:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.startState,
			}
			assert.Equal(t, tt.wantedResult, ms.IsMetricInStorage(tt.metricToCheck))
		})
	}
}

func TestMemStorage_UpdateMetric(t *testing.T) {
	tests := []struct {
		name           string
		metricToUpdate Metric
		newValue       interface{}
		startState     map[string]Metric
		wantedState    map[string]Metric
		wantError      error
	}{
		{
			name:           "Test 1. Update metric, type 'counter'",
			metricToUpdate: testMetric11,
			newValue:       counter(15),
			startState:     map[string]Metric{testMetric11.Name: testMetric11},
			wantedState:    map[string]Metric{"testMetric1": {Name: "testMetric1", Value: counter(25)}},
			wantError:      nil,
		},
		{
			name:           "Test 2. Update metric, type 'gauge'",
			metricToUpdate: testMetric13,
			newValue:       gauge(27.3),
			startState:     map[string]Metric{testMetric13.Name: testMetric13},
			wantedState:    map[string]Metric{"testMetric1": {Name: "testMetric1", Value: gauge(27.3)}},
			wantError:      nil,
		},
		{
			name:           "Test 3. Update metric, wrong type 'gauge'",
			metricToUpdate: testMetric11,
			newValue:       gauge(27.3),
			startState:     map[string]Metric{testMetric11.Name: testMetric11},
			wantedState:    map[string]Metric{testMetric11.Name: testMetric11},
			wantError: fmt.Errorf("updated(%s) and new(%s) value type mismatch",
				"storage.counter", "storage.gauge"),
		},
		{
			name:           "Test 8. Empty state",
			metricToUpdate: testMetric11,
			newValue:       counter(15),
			startState:     map[string]Metric{},
			wantedState:    map[string]Metric{},
			wantError:      fmt.Errorf("there is no metric with name '%s'", testMetric11.Name),
		},
		{
			name:           "Test 9. Metric to update is not present",
			metricToUpdate: testMetric11,
			newValue:       counter(15),
			startState:     map[string]Metric{testMetric4.Name: testMetric4},
			wantedState:    map[string]Metric{testMetric4.Name: testMetric4},
			wantError:      fmt.Errorf("there is no metric with name '%s'", testMetric11.Name),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.startState,
			}
			tt.metricToUpdate.Value = tt.newValue
			err := ms.UpdateMetric(tt.metricToUpdate)
			assert.Equal(t, tt.wantedState, ms.metrics)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

// Упрощенная версия теста, без дублирования тестирования методов IsMetricInStorage ->
// -> AddMetric и UpdateMetric, на которых эта функция основана.
func TestMemStorage_UpdateOrAddMetric(t *testing.T) {

	tests := []struct {
		name        string
		metricObj   Metric
		startState  map[string]Metric
		wantedState map[string]Metric
	}{
		{
			name:       "Test 1. Add new metric.",
			metricObj:  testMetric4,
			startState: map[string]Metric{testMetric11.Name: testMetric11},
			wantedState: map[string]Metric{
				testMetric11.Name: testMetric11,
				testMetric4.Name:  testMetric4,
			},
		},
		{
			name:      "Test 2. Update existed metric.",
			metricObj: testMetric12,
			startState: map[string]Metric{
				testMetric11.Name: testMetric11,
				testMetric4.Name:  testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric11.Name: {Name: testMetric11.Name, Value: counter(25)},
				testMetric4.Name:  testMetric4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.startState,
			}
			_ = ms.UpdateOrAddMetric(tt.metricObj)
			assert.Equal(t, tt.wantedState, ms.metrics)
		})
	}
}

func TestMemStorage_GetAll(t *testing.T) {
	tests := []struct {
		name  string
		state map[string]Metric
		want  map[string]Metric
	}{
		{
			name:  "Test 1. Empty state.",
			state: map[string]Metric{},
			want:  map[string]Metric{},
		},
		{
			name: "Test 2. State contains metrics.",
			state: map[string]Metric{
				testMetric11.Name: testMetric11,
				testMetric4.Name:  testMetric4,
			},
			want: map[string]Metric{
				testMetric11.Name: testMetric11,
				testMetric4.Name:  testMetric4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.state,
			}
			gotMapMetrics := ms.GetAll()
			assert.Equal(t, tt.want, gotMapMetrics)
		})
	}
}

func TestMemStorage_GetMetric(t *testing.T) {
	tests := []struct {
		name       string
		state      map[string]Metric
		metricName string
		wantMetric Metric
		wantOk     bool
	}{
		{
			name:       "Test 1. State contains requested metric.",
			state:      map[string]Metric{testMetric11.Name: testMetric11},
			metricName: testMetric11.Name,
			wantMetric: testMetric11,
			wantOk:     true,
		},
		{
			name:       "Test 2. State doesn't contain requested metric.",
			state:      map[string]Metric{testMetric4.Name: testMetric4},
			metricName: testMetric11.Name,
			wantMetric: Metric{},
			wantOk:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.state,
			}
			gotMetric, gotOk := ms.GetMetric(tt.metricName)
			require.Equal(t, tt.wantOk, gotOk)
			assert.Equal(t, tt.wantMetric, gotMetric)
		})
	}
}

func TestNewMemStorage(t *testing.T) {
	tests := []struct {
		name       string
		argMetrics map[string]Metric
		want       MemStorage
	}{
		{
			name:       "Test 1. Not nil arg metrics.",
			argMetrics: map[string]Metric{},
			want:       MemStorage{metrics: map[string]Metric{}},
		},
		{
			name:       "Test 2. Nil arg metrics.",
			argMetrics: nil,
			want:       MemStorage{metrics: nil},
		},
		{
			name: "Test 3. Arg metrics filled with metrics.",
			argMetrics: map[string]Metric{
				testMetric11.Name: testMetric11,
				testMetric4.Name:  testMetric4,
			},
			want: MemStorage{
				metrics: map[string]Metric{
					testMetric11.Name: testMetric11,
					testMetric4.Name:  testMetric4,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			memStorageObj := *NewMemStorage(tt.argMetrics)
			assert.Equal(t, tt.want, memStorageObj)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMetric, err := NewMetric(tt.args.name, tt.args.typeName, tt.args.rawValue)
			assert.Equal(t, tt.want, gotMetric)
			assert.Equal(t, tt.wantError, err)
		})
	}
}
