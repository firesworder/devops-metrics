package storage

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

var metric1counter20, _ = NewMetric("testMetric1", "counter", int64(20))

var devDSN = "postgresql://postgres:admin@localhost:5432/devops"

func TestSqlStorage_BatchUpdate(t *testing.T) {
	var err error
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name         string
		metricsBatch map[string]Metric
		initDBState  map[string]Metric
		wantDBState  map[string]Metric
	}{
		{
			name: "Test 1. First batch(empty table metrics)",
			metricsBatch: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
		},
		{
			name: "Test 2. Partially update, partially add(table has some of metrics from batch)",
			metricsBatch: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			initDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
			},
			wantDBState: map[string]Metric{
				metric1Counter10.Name: *metric1counter20,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
		},
		{
			name: "Test 3. Only update(table metrics contains all batch metrics)",
			metricsBatch: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			initDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			wantDBState: map[string]Metric{
				metric1Counter10.Name: *metric1counter20,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// подготовка состояния таблицы
			_, err = sqlStorage.Connection.Exec("DELETE FROM metrics")
			require.NoError(t, err)
			for _, metric := range tt.initDBState {
				err = sqlStorage.AddMetric(metric)
				require.NoError(t, err)
			}

			// сам тест
			err = sqlStorage.BatchUpdate(tt.metricsBatch)
			require.NoError(t, err)

			gotDBState := sqlStorage.GetAll()
			assert.Equal(t, tt.wantDBState, gotDBState)
		})
	}
}
