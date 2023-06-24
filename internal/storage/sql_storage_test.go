package storage

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var metric1counter20, _ = NewMetric("testMetric1", "counter", int64(20))

var devDSN = "postgresql://postgres:admin@localhost:5432/devops"

func prepareDBState(t *testing.T, s *SQLStorage, ctx context.Context, wantDBState map[string]Metric) {
	// подготовка состояния таблицы
	_, err := s.Connection.ExecContext(ctx, "DELETE FROM metrics")
	require.NoError(t, err)
	for _, metric := range wantDBState {
		mN, mV, mT := metric.GetMetricParamsString()
		_, err = s.Connection.ExecContext(ctx,
			"INSERT INTO metrics(m_name, m_value, m_type) VALUES($1, $2, $3)", mN, mV, mT)
		require.NoError(t, err)
	}
}

func TestSqlStorage_BatchUpdate(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name         string
		metricsBatch []Metric
		initDBState  map[string]Metric
		wantDBState  map[string]Metric
	}{
		{
			name: "Test 1. First batch(empty table metrics)",
			metricsBatch: []Metric{
				metric1Counter10,
				metric4Gauge2d27,
			},
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
		},
		{
			name: "Test 2. Partially update, partially add(table has some of metrics from batch)",
			metricsBatch: []Metric{
				metric1Counter10,
				metric4Gauge2d27,
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
			metricsBatch: []Metric{
				metric1Counter10,
				metric4Gauge2d27,
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
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			err = sqlStorage.BatchUpdate(ctx, tt.metricsBatch)
			require.NoError(t, err)

			gotDBState, err := sqlStorage.GetAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDBState, gotDBState)
		})
	}
}

func TestSQLStorage_AddMetric(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name        string
		metric      Metric
		initDBState map[string]Metric
		wantDBState map[string]Metric
		wantError   bool
	}{
		{
			name:        "Test 1. Metric not present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{metric1Counter10.Name: metric1Counter10},
			wantError:   false,
		},
		{
			name:        "Test 2. Metric not present in db. Gauge type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{metric4Gauge2d27.Name: metric4Gauge2d27},
			wantError:   false,
		},
		{
			name:        "Test 3. Metric already present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{metric1Counter10.Name: metric1Counter10},
			wantDBState: map[string]Metric{metric1Counter10.Name: metric1Counter10},
			wantError:   true,
		},
		{
			name:        "Test 4. Metric already present in db. Counter type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{metric4Gauge2d27.Name: metric4Gauge2d27},
			wantDBState: map[string]Metric{metric4Gauge2d27.Name: metric4Gauge2d27},
			wantError:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			err = sqlStorage.AddMetric(ctx, tt.metric)
			assert.Equal(t, tt.wantError, err != nil)

			gotDBState, err := sqlStorage.GetAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDBState, gotDBState)
		})
	}
}

func TestSQLStorage_UpdateMetric(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name        string
		metric      Metric
		initDBState map[string]Metric
		wantDBState map[string]Metric
		wantError   bool
	}{
		{
			name:        "Test 1. Metric not present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{},
			wantError:   true,
		},
		{
			name:        "Test 2. Metric not present in db. Gauge type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{},
			wantError:   true,
		},
		{
			name:        "Test 3. Metric present in db. Counter type",
			metric:      Metric{Name: metric1Counter10.Name, Value: counter(15)},
			initDBState: map[string]Metric{metric1Counter10.Name: metric1Counter10},
			wantDBState: map[string]Metric{
				metric1Counter10.Name: {Name: metric1Counter10.Name, Value: counter(25)},
			},
			wantError: false,
		},
		{
			name:        "Test 4. Metric already present in db. Counter type",
			metric:      Metric{Name: metric4Gauge2d27.Name, Value: gauge(12.133)},
			initDBState: map[string]Metric{metric4Gauge2d27.Name: metric4Gauge2d27},
			wantDBState: map[string]Metric{
				metric4Gauge2d27.Name: Metric{Name: metric4Gauge2d27.Name, Value: gauge(12.133)},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			err = sqlStorage.UpdateMetric(ctx, tt.metric)
			assert.Equal(t, tt.wantError, err != nil)

			gotDBState, err := sqlStorage.GetAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDBState, gotDBState)
		})
	}
}

func TestSQLStorage_DeleteMetric(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name        string
		metric      Metric
		initDBState map[string]Metric
		wantDBState map[string]Metric
		wantError   bool
	}{
		{
			name:        "Test 1. Metric not present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{},
			wantError:   true,
		},
		{
			name:        "Test 2. Metric not present in db. Gauge type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{},
			wantError:   true,
		},
		{
			name:        "Test 3. Metric present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{metric1Counter10.Name: metric1Counter10},
			wantDBState: map[string]Metric{},
			wantError:   false,
		},
		{
			name:        "Test 4. Metric already present in db. Counter type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{metric4Gauge2d27.Name: metric4Gauge2d27},
			wantDBState: map[string]Metric{},
			wantError:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			err = sqlStorage.DeleteMetric(ctx, tt.metric)
			assert.Equal(t, tt.wantError, err != nil)

			gotDBState, err := sqlStorage.GetAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDBState, gotDBState)
		})
	}
}

func TestSQLStorage_IsMetricInStorage(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name        string
		metric      Metric
		initDBState map[string]Metric
		isExist     bool
		wantError   bool
	}{
		{
			name:        "Test 1. Metric not present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{},
			isExist:     false,
			wantError:   false,
		},
		{
			name:        "Test 2. Metric not present in db. Gauge type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{},
			isExist:     false,
			wantError:   false,
		},
		{
			name:        "Test 3. Metric present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{metric1Counter10.Name: metric1Counter10},
			isExist:     true,
			wantError:   false,
		},
		{
			name:        "Test 4. Metric already present in db. Counter type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{metric4Gauge2d27.Name: metric4Gauge2d27},
			isExist:     true,
			wantError:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			isExist, err := sqlStorage.IsMetricInStorage(ctx, tt.metric)
			assert.Equal(t, tt.isExist, isExist)
			assert.Equal(t, tt.wantError, err != nil)
		})
	}
}

func TestSQLStorage_UpdateOrAddMetric(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name        string
		metric      Metric
		initDBState map[string]Metric
		wantDBState map[string]Metric
		wantError   bool
	}{
		{
			name:        "Test 1. Metric not present in db. Counter type",
			metric:      metric1Counter10,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{metric1Counter10.Name: metric1Counter10},
			wantError:   false,
		},
		{
			name:        "Test 2. Metric not present in db. Gauge type",
			metric:      metric4Gauge2d27,
			initDBState: map[string]Metric{},
			wantDBState: map[string]Metric{metric4Gauge2d27.Name: metric4Gauge2d27},
			wantError:   false,
		},
		{
			name:   "Test 3. Metric present in db. Counter type",
			metric: Metric{Name: metric1Counter10.Name, Value: counter(15)},
			initDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			wantDBState: map[string]Metric{
				metric1Counter10.Name: {Name: metric1Counter10.Name, Value: counter(25)},
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			wantError: false,
		},
		{
			name:   "Test 4. Metric already present in db. Counter type",
			metric: Metric{Name: metric4Gauge2d27.Name, Value: gauge(12.133)},
			initDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			wantDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: Metric{Name: metric4Gauge2d27.Name, Value: gauge(12.133)},
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			err = sqlStorage.UpdateOrAddMetric(ctx, tt.metric)
			assert.Equal(t, tt.wantError, err != nil)

			gotDBState, err := sqlStorage.GetAll(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.wantDBState, gotDBState)
		})
	}
}

func TestSQLStorage_GetAll(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name        string
		initDBState map[string]Metric
		wantResult  Metric
		wantError   bool
	}{
		{
			name:        "Test 1. Empty state",
			initDBState: map[string]Metric{},
			wantError:   false,
		},
		{
			name: "Test 2. Filled state",
			initDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			wantError: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			gotAllMetrics, err := sqlStorage.GetAll(ctx)
			assert.Equal(t, tt.wantError, err != nil)
			assert.Equal(t, tt.initDBState, gotAllMetrics)
		})
	}
}

func TestSQLStorage_GetMetric(t *testing.T) {
	var err error
	ctx := context.Background()
	sqlStorage, err := NewSQLStorage(devDSN)
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	defer sqlStorage.Connection.Close()

	tests := []struct {
		name        string
		metricName  string
		initDBState map[string]Metric
		wantMetric  Metric
		wantError   error
	}{
		{
			name:        "Test 1. Metric not present in db. Counter type",
			metricName:  metric1Counter10.Name,
			initDBState: map[string]Metric{},
			wantMetric:  Metric{},
			wantError:   ErrMetricNotFound,
		},
		{
			name:        "Test 2. Metric not present in db. Gauge type",
			metricName:  metric4Gauge2d27.Name,
			initDBState: map[string]Metric{},
			wantMetric:  Metric{},
			wantError:   ErrMetricNotFound,
		},
		{
			name:       "Test 3. Metric present in db. Counter type",
			metricName: metric1Counter10.Name,
			initDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			wantMetric: metric1Counter10,
			wantError:  nil,
		},
		{
			name:       "Test 4. Metric already present in db. Counter type",
			metricName: metric4Gauge2d27.Name,
			initDBState: map[string]Metric{
				metric1Counter10.Name: metric1Counter10,
				metric4Gauge2d27.Name: metric4Gauge2d27,
			},
			wantMetric: metric4Gauge2d27,
			wantError:  nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prepareDBState(t, sqlStorage, ctx, tt.initDBState)

			metric, err := sqlStorage.GetMetric(ctx, tt.metricName)
			assert.ErrorIs(t, err, tt.wantError)
			assert.Equal(t, tt.wantMetric, metric)
		})
	}
}
