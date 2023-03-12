package storage

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMemStorage_AddMetric(t *testing.T) {
	testMetric11 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	testMetric12 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	testMetric13 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	testMetric5 := Metric{name: "testMetric5", value: 0}
	testMetric6 := Metric{}

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
			wantedState: map[string]Metric{testMetric11.name: testMetric11},
			wantError:   nil,
		},
		{
			name:        "Test 2. Add metric to storage, but metric already present.",
			metricToAdd: testMetric12,
			startState:  map[string]Metric{testMetric11.name: testMetric11},
			wantedState: map[string]Metric{testMetric11.name: testMetric11},
			wantError:   fmt.Errorf("metric with name '%s' already present in Storage", testMetric12.name),
		},
		{
			name:        "Test 3. Add metric to storage, but metric already present. Value type differ",
			metricToAdd: testMetric13,
			startState:  map[string]Metric{testMetric11.name: testMetric11},
			wantedState: map[string]Metric{testMetric11.name: testMetric11},
			wantError:   fmt.Errorf("metric with name '%s' already present in Storage", testMetric13.name),
		},
		{
			name:        "Test 4. Add another metric to storage",
			metricToAdd: testMetric4,
			startState:  map[string]Metric{testMetric11.name: testMetric11},
			wantedState: map[string]Metric{testMetric11.name: testMetric11, testMetric4.name: testMetric4},
			wantError:   nil,
		},
		{
			name:        "Test 5. Add metric with unhandled value type",
			metricToAdd: testMetric5,
			startState:  map[string]Metric{},
			wantedState: map[string]Metric{},
			wantError:   fmt.Errorf("unhandled value type '%T'", testMetric5.value),
		},
		{
			name:        "Test 6. Add empty metric",
			metricToAdd: testMetric6,
			startState:  map[string]Metric{},
			wantedState: map[string]Metric{},
			wantError:   fmt.Errorf("unhandled value type '%T'", testMetric6.value),
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
	testMetric1 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	//testMetric2 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	testMetric3 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	testMetric5 := Metric{name: "testMetric5", value: 0}
	testMetric6 := Metric{}
	testMetric7 := Metric{name: "testMetric7", value: counter(27)}

	tests := []struct {
		name           string
		metricToDelete Metric
		startState     map[string]Metric
		wantedState    map[string]Metric
	}{
		{
			name:           "Test 1. Delete metric from state contains ONLY that metric.",
			metricToDelete: testMetric1,
			startState:     map[string]Metric{testMetric1.name: testMetric1},
			wantedState:    map[string]Metric{},
		},
		{
			name:           "Test 2. Delete metric from state contains that metric.",
			metricToDelete: testMetric1,
			startState: map[string]Metric{
				testMetric1.name: testMetric1,
				testMetric4.name: testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric4.name: testMetric4,
			},
		},
		{
			name:           "Test 3. Delete metric from state contains metrics, except that metric.",
			metricToDelete: testMetric1,
			startState: map[string]Metric{
				testMetric7.name: testMetric7,
				testMetric4.name: testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric7.name: testMetric7,
				testMetric4.name: testMetric4,
			},
		},
		{
			name:           "Test 4. Delete metric from state contains that metric, but value differ.",
			metricToDelete: testMetric1,
			startState: map[string]Metric{
				testMetric3.name: testMetric3,
				testMetric4.name: testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric4.name: testMetric4,
			},
		},
		{
			name:           "Test 5. Delete metric from empty state.",
			metricToDelete: testMetric1,
			startState:     map[string]Metric{},
			wantedState:    map[string]Metric{},
		},
		{
			name:           "Test 6. Delete empty metric.",
			metricToDelete: testMetric6,
			startState: map[string]Metric{
				testMetric6.name: testMetric6,
				testMetric7.name: testMetric7,
			},
			wantedState: map[string]Metric{testMetric7.name: testMetric7},
		},
		{
			name:           "Test 7. Delete metric with unhandled value type.",
			metricToDelete: testMetric5,
			startState: map[string]Metric{
				testMetric5.name: testMetric5,
				testMetric7.name: testMetric7,
			},
			wantedState: map[string]Metric{testMetric7.name: testMetric7},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.startState,
			}
			ms.DeleteMetric(tt.metricToDelete)
			assert.Equal(t, tt.wantedState, ms.metrics)
		})
	}
}

func TestMemStorage_IsMetricInStorage(t *testing.T) {
	testMetric1 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	//testMetric2 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	testMetric3 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	//testMetric5 := Metric{name: "testMetric5", value: 0}
	testMetric6 := Metric{}
	testMetric7 := Metric{name: "testMetric7", value: counter(27)}

	tests := []struct {
		name          string
		metricToCheck Metric
		startState    map[string]Metric
		wantedResult  bool
	}{
		{
			name:          "Test 1. Searched metric present in state. State contains only that metric.",
			metricToCheck: testMetric1,
			startState:    map[string]Metric{testMetric1.name: testMetric1},
			wantedResult:  true,
		},
		{
			name:          "Test 2. Searched metric present in state. Multiple metrics in state.",
			metricToCheck: testMetric1,
			startState:    map[string]Metric{testMetric1.name: testMetric1, testMetric4.name: testMetric4},
			wantedResult:  true,
		},
		{
			name:          "Test 3. Metric name present in state, but value differs",
			metricToCheck: testMetric1,
			startState:    map[string]Metric{testMetric3.name: testMetric3, testMetric4.name: testMetric4},
			wantedResult:  true,
		},
		{
			name:          "Test 4. Metric is not present in state.",
			metricToCheck: testMetric1,
			startState:    map[string]Metric{testMetric7.name: testMetric7, testMetric4.name: testMetric4},
			wantedResult:  false,
		},
		{
			name:          "Test 5. Empty state.",
			metricToCheck: testMetric1,
			startState:    map[string]Metric{},
			wantedResult:  false,
		},
		{
			name:          "Test 5. Empty metric.",
			metricToCheck: testMetric6,
			startState:    map[string]Metric{testMetric6.name: testMetric6, testMetric4.name: testMetric4},
			wantedResult:  true,
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
	testMetric11 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	//testMetric12 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	testMetric13 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	//testMetric5 := Metric{name: "testMetric5", value: 0}
	//testMetric6 := Metric{}
	//testMetric7 := Metric{name: "testMetric7", value: counter(27)}

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
			startState:     map[string]Metric{testMetric11.name: testMetric11},
			wantedState:    map[string]Metric{"testMetric1": {name: "testMetric1", value: counter(25)}},
			wantError:      nil,
		},
		{
			name:           "Test 2. Update metric, type 'gauge'",
			metricToUpdate: testMetric13,
			newValue:       gauge(27.3),
			startState:     map[string]Metric{testMetric13.name: testMetric13},
			wantedState:    map[string]Metric{"testMetric1": {name: "testMetric1", value: gauge(27.3)}},
			wantError:      nil,
		},
		{
			name:           "Test 3. Update metric, wrong type 'gauge'",
			metricToUpdate: testMetric11,
			newValue:       gauge(27.3),
			startState:     map[string]Metric{testMetric11.name: testMetric11},
			wantedState:    map[string]Metric{testMetric11.name: testMetric11},
			wantError: fmt.Errorf("updated(%s) and new(%s) value type mismatch",
				"storage.counter", "storage.gauge"),
		},
		{
			name:           "Test 8. Empty state",
			metricToUpdate: testMetric11,
			newValue:       counter(15),
			startState:     map[string]Metric{},
			wantedState:    map[string]Metric{},
			wantError:      fmt.Errorf("there is no metric with name '%s'", testMetric11.name),
		},
		{
			name:           "Test 9. Metric to update is not present",
			metricToUpdate: testMetric11,
			newValue:       counter(15),
			startState:     map[string]Metric{testMetric4.name: testMetric4},
			wantedState:    map[string]Metric{testMetric4.name: testMetric4},
			wantError:      fmt.Errorf("there is no metric with name '%s'", testMetric11.name),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.startState,
			}
			tt.metricToUpdate.value = tt.newValue
			err := ms.UpdateMetric(tt.metricToUpdate)
			assert.Equal(t, tt.wantedState, ms.metrics)
			assert.Equal(t, tt.wantError, err)
		})
	}
}

// Упрощенная версия теста, без дублирования тестирования методов IsMetricInStorage ->
// -> AddMetric и UpdateMetric, на которых эта функция основана.
func TestMemStorage_UpdateOrAddMetric(t *testing.T) {
	testMetric11 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	testMetric12 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	//testMetric13 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	//testMetric5 := Metric{name: "testMetric5", value: 0}
	//testMetric6 := Metric{}
	//testMetric7 := Metric{name: "testMetric7", value: counter(27)}

	tests := []struct {
		name        string
		metricObj   Metric
		startState  map[string]Metric
		wantedState map[string]Metric
	}{
		{
			name:       "Test 1. Add new metric.",
			metricObj:  testMetric4,
			startState: map[string]Metric{testMetric11.name: testMetric11},
			wantedState: map[string]Metric{
				testMetric11.name: testMetric11,
				testMetric4.name:  testMetric4,
			},
		},
		{
			name:      "Test 2. Update existed metric.",
			metricObj: testMetric12,
			startState: map[string]Metric{
				testMetric11.name: testMetric11,
				testMetric4.name:  testMetric4,
			},
			wantedState: map[string]Metric{
				testMetric11.name: {name: testMetric11.name, value: counter(25)},
				testMetric4.name:  testMetric4,
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

func TestMemStorage_getAll(t *testing.T) {
	testMetric11 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	//testMetric12 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	//testMetric13 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	//testMetric5 := Metric{name: "testMetric5", value: 0}
	//testMetric6 := Metric{}
	//testMetric7 := Metric{name: "testMetric7", value: counter(27)}

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
				testMetric11.name: testMetric11,
				testMetric4.name:  testMetric4,
			},
			want: map[string]Metric{
				testMetric11.name: testMetric11,
				testMetric4.name:  testMetric4,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.state,
			}
			assert.Equal(t, tt.want, ms.metrics)
		})
	}
}

func TestMemStorage_getMetric(t *testing.T) {
	testMetric11 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	//testMetric12 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	//testMetric13 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	//testMetric5 := Metric{name: "testMetric5", value: 0}
	//testMetric6 := Metric{}
	//testMetric7 := Metric{name: "testMetric7", value: counter(27)}

	tests := []struct {
		name       string
		state      map[string]Metric
		metricName string
		wantMetric Metric
		wantOk     bool
	}{
		{
			name:       "Test 1. State contains requested metric.",
			state:      map[string]Metric{testMetric11.name: testMetric11},
			metricName: testMetric11.name,
			wantMetric: testMetric11,
			wantOk:     true,
		},
		{
			name:       "Test 2. State doesn't contain requested metric.",
			state:      map[string]Metric{testMetric4.name: testMetric4},
			metricName: testMetric11.name,
			wantMetric: Metric{},
			wantOk:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &MemStorage{
				metrics: tt.state,
			}
			gotMetric, gotOk := ms.getMetric(tt.metricName)
			require.Equal(t, tt.wantOk, gotOk)
			assert.Equal(t, tt.wantMetric, gotMetric)
		})
	}
}

func TestNewMemStorage(t *testing.T) {

	testMetric11 := Metric{name: "testMetric1", value: counter(10)}
	// одинаковый name с testMetric1, но другое value
	//testMetric12 := Metric{name: "testMetric1", value: counter(15)}
	// одинаковый name с testMetric1, но другое value и тип value
	//testMetric13 := Metric{name: "testMetric1", value: gauge(22.2)}
	testMetric4 := Metric{name: "testMetric4", value: gauge(2.27)}
	//testMetric5 := Metric{name: "testMetric5", value: 0}
	//testMetric6 := Metric{}
	//testMetric7 := Metric{name: "testMetric7", value: counter(27)}

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
				testMetric11.name: testMetric11,
				testMetric4.name:  testMetric4,
			},
			want: MemStorage{
				metrics: map[string]Metric{
					testMetric11.name: testMetric11,
					testMetric4.name:  testMetric4,
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
		name string
		args args
		want Metric
	}{
		{
			name: "Test 1. Counter type metric, with correct value.",
			args: args{name: "testMetric11", typeName: "counter", rawValue: int64(10)},
			want: Metric{name: "testMetric11", value: counter(10)},
		},
		// todo: тут ошибка должна быть
		{
			name: "Test 2. Counter type metric, with incorrect number value.",
			args: args{name: "testMetric12", typeName: "counter", rawValue: 11.3},
			want: Metric{},
		},
		// todo: тут ошибка должна быть
		{
			name: "Test 3. Counter type metric, with incorrect NAN value.",
			args: args{name: "testMetric13", typeName: "counter", rawValue: "str"},
			want: Metric{},
		},
		{
			name: "Test 4. Gauge type metric, with correct value.",
			args: args{name: "testMetric2", typeName: "gauge", rawValue: 11.2},
			want: Metric{name: "testMetric2", value: gauge(11.2)},
		},
		{
			name: "Test 5. Gauge type metric, with incorrect number value.",
			args: args{name: "testMetric2", typeName: "gauge", rawValue: 10},
			want: Metric{},
		},
		{
			name: "Test 6. Gauge type metric, with incorrect NAN value.",
			args: args{name: "testMetric2", typeName: "gauge", rawValue: "str"},
			want: Metric{},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMetric := *NewMetric(tt.args.name, tt.args.typeName, tt.args.rawValue)
			assert.Equal(t, tt.want, gotMetric)
		})
	}
}
