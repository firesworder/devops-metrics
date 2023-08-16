package server

import (
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/server/env"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

var defaultEnv = env.Environment{
	ServerAddress: "localhost:8080",
	StoreFile:     "/tmp/devops-metrics-db.json",
	Restore:       true,
	StoreInterval: 300 * time.Second,
}

var metricCounterPC10, _ = storage.NewMetric("PollCount", internal.CounterTypeName, int64(10))
var metricGaugeRV12p23, _ = storage.NewMetric("RandomValue", internal.GaugeTypeName, float64(12.23))
var metricCounterPC25, _ = storage.NewMetric("PollCount", internal.CounterTypeName, int64(25))

// todo: описать тесты в примитивной форме

func TestNewServer(t *testing.T) {
	// default env
	envTest := defaultEnv
	envTest.DatabaseDsn = "postgresql://postgres:admin@localhost:5432/devops"

	s, err := NewServer(&envTest)
	require.NoError(t, err)
	assert.Equal(t, true, s.dbConn != nil)
}
