package server

import (
	"context"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGetAllMetrics(t *testing.T) {
	testEnv := defaultEnv
	s, err := NewServer(&testEnv)
	require.NoError(t, err)

	s.metricStorage = &storage.MemStorage{Metrics: map[string]storage.Metric{
		metricCounterPC10.Name:  *metricCounterPC10,
		metricGaugeRV12p23.Name: *metricGaugeRV12p23,
	}}

	metrics, err := s.GetAllMetrics(context.Background())
	require.NoError(t, err)
	assert.Contains(t, metrics, metricCounterPC10.Name)
	assert.Contains(t, metrics, metricGaugeRV12p23.Name)
	assert.Equal(t, 2, len(metrics))
}

func TestPing(t *testing.T) {
	testEnv := defaultEnv
	// todo: убрать связь с тестовой дб
	testEnv.DatabaseDsn = "postgresql://postgres:admin@localhost:5432/devops"
	s, err := NewServer(&testEnv)
	require.NoError(t, err)
	require.NoError(t, s.Ping(context.Background()))
}

func TestUpdateMetric(t *testing.T) {
	testKey := "Ayayaka"

	testEnv := defaultEnv
	testEnv.Key = testKey
	s, err := NewServer(&testEnv)
	require.NoError(t, err)

	s.metricStorage = &storage.MemStorage{Metrics: map[string]storage.Metric{
		metricCounterPC10.Name:  *metricCounterPC10,
		metricGaugeRV12p23.Name: *metricGaugeRV12p23,
	}}

	intVI, intVW := int64(20), int64(30)
	msgCounter10 := message.Metrics{
		ID:    "PollCount",
		MType: internal.CounterTypeName,
		Delta: &intVI,
	}
	require.NoError(t, msgCounter10.InitHash(testKey))
	wantResCounter30 := message.Metrics{
		ID:    "PollCount",
		MType: internal.CounterTypeName,
		Delta: &intVW,
	}
	require.NoError(t, wantResCounter30.InitHash(testKey))

	msgC, err := s.UpdateMetric(context.Background(), msgCounter10)
	require.NoError(t, err)
	assert.Equal(t, wantResCounter30, *msgC)

	// float64(отправленное сообщение и ответ для gauge - идентичны)
	floatVI := float64(23.5)
	msgGauge23p5 := message.Metrics{
		ID:    "RandomValue",
		MType: internal.GaugeTypeName,
		Value: &floatVI,
	}
	require.NoError(t, msgGauge23p5.InitHash(testKey))

	msgG, err := s.UpdateMetric(context.Background(), msgGauge23p5)
	require.NoError(t, err)
	assert.Equal(t, msgGauge23p5, *msgG)
}

func TestGetMetric(t *testing.T) {
	testKey := "Ayayaka"

	testEnv := defaultEnv
	testEnv.Key = testKey
	s, err := NewServer(&testEnv)
	require.NoError(t, err)

	s.metricStorage = &storage.MemStorage{Metrics: map[string]storage.Metric{
		metricCounterPC10.Name:  *metricCounterPC10,
		metricGaugeRV12p23.Name: *metricGaugeRV12p23,
	}}

	msgGauge23p5 := message.Metrics{ID: "RandomValue"}

	msgG, err := s.GetMetric(context.Background(), msgGauge23p5)
	require.NoError(t, err)
	wantRes := metricGaugeRV12p23.GetMessageMetric()
	wantRes.InitHash(testKey)
	assert.Equal(t, wantRes, *msgG)
}

func TestBatchUpdate(t *testing.T) {
	testKey := "Ayayaka"

	testEnv := defaultEnv
	testEnv.Key = testKey
	s, err := NewServer(&testEnv)
	require.NoError(t, err)

	memStorage := &storage.MemStorage{Metrics: map[string]storage.Metric{
		metricCounterPC10.Name:  *metricCounterPC10,
		metricGaugeRV12p23.Name: *metricGaugeRV12p23,
	}}
	s.metricStorage = memStorage

	intVI, intVW := int64(20), int64(30)
	msgCounter10 := message.Metrics{
		ID:    "PollCount",
		MType: internal.CounterTypeName,
		Delta: &intVI,
	}
	require.NoError(t, msgCounter10.InitHash(testKey))
	wantResCounter30 := message.Metrics{
		ID:    "PollCount",
		MType: internal.CounterTypeName,
		Delta: &intVW,
	}
	require.NoError(t, wantResCounter30.InitHash(testKey))

	// float64(отправленное сообщение и ответ для gauge - идентичны)
	floatVI := float64(23.5)
	msgGauge23p5 := message.Metrics{
		ID:    "RandomValue",
		MType: internal.GaugeTypeName,
		Value: &floatVI,
	}
	require.NoError(t, msgGauge23p5.InitHash(testKey))

	metricMessagesBatch := []message.Metrics{
		msgCounter10,
		msgGauge23p5,
	}

	ctx := context.Background()
	err = s.BatchUpdate(ctx, metricMessagesBatch)
	require.NoError(t, err)

	wantC, _ := storage.NewMetric("PollCount", internal.CounterTypeName, int64(30))
	assert.Equal(t, memStorage.Metrics["PollCount"], *wantC)
	wantG, _ := storage.NewMetric("RandomValue", internal.GaugeTypeName, float64(23.5))
	assert.Equal(t, memStorage.Metrics["RandomValue"], *wantG)
}
