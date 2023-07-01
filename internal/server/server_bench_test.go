package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
)

const repeatBenchRun = 100

func BenchmarkHandlersMemStorage(b *testing.B) {
	s := getServer(false)
	ts := httptest.NewServer(s.Router)
	defer ts.Close()

	b.Run("getMetric", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodGet, ts.URL+"/value/counter/PollCount", "text/plain", "")
		}
	})

	b.Run("showAllMetrics", func(b *testing.B) {
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodGet, ts.URL+"/", "text/plain", "")
		}
	})

	b.Run("AddUpdateMetric", func(b *testing.B) {
		urlParams := `/update/counter/PollCount/20`
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+urlParams, "text/plain", "")

			b.StopTimer()
			s.MetricStorage = storage.NewMemStorage(getMetricsMap())
			b.StartTimer()
		}
	})

	b.Run("JSONAddUpdateMetric", func(b *testing.B) {
		metric, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(15))
		jsonMsg, _ := json.Marshal(metric.GetMessageMetric())
		url := "/update/"
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))

			b.StopTimer()
			s.MetricStorage = storage.NewMemStorage(getMetricsMap())
			b.StartTimer()
		}
	})

	b.Run("JSONGetMetric", func(b *testing.B) {
		jsonMsg, _ := json.Marshal(message.Metrics{ID: "PollCount"})
		url := "/value/"
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))
		}
	})

	b.Run("BatchUpdate", func(b *testing.B) {
		m1, _ := storage.NewMetric("PollCount", internal.CounterTypeName, int64(40))
		m2, _ := storage.NewMetric("Alloc", internal.GaugeTypeName, 13.345)
		m3, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(99))
		msgSlice := []message.Metrics{
			m1.GetMessageMetric(), m2.GetMessageMetric(), m3.GetMessageMetric(),
		}
		jsonMsg, _ := json.Marshal(msgSlice)

		url := "/updates/"
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))

			b.StopTimer()
			s.MetricStorage = storage.NewMemStorage(getMetricsMap())
			b.StartTimer()
		}
	})
}

func BenchmarkHandlersSQLStorage(b *testing.B) {
	s := getServer(true)
	ts := httptest.NewServer(s.Router)
	defer ts.Close()

	fmt.Println(b.N)

	b.Run("getMetric", func(b *testing.B) {
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodGet, ts.URL+"/value/counter/PollCount", "text/plain", "")
		}
	})

	b.Run("showAllMetrics", func(b *testing.B) {
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodGet, ts.URL+"/", "text/plain", "")
		}
	})

	b.Run("AddUpdateMetric", func(b *testing.B) {
		urlParams := `/update/counter/PollCount/20`
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+urlParams, "text/plain", "")

			b.StopTimer()
			resetDBState(s.DBConn)
			b.StartTimer()
		}
	})

	b.Run("JSONAddUpdateMetric", func(b *testing.B) {
		metric, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(15))
		jsonMsg, _ := json.Marshal(metric.GetMessageMetric())
		url := "/update/"
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))

			b.StopTimer()
			resetDBState(s.DBConn)
			b.StartTimer()
		}
	})

	b.Run("JSONGetMetric", func(b *testing.B) {
		jsonMsg, _ := json.Marshal(message.Metrics{ID: "PollCount"})
		url := "/value/"
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))
		}
	})

	b.Run("BatchUpdate", func(b *testing.B) {
		m1, _ := storage.NewMetric("PollCount", internal.CounterTypeName, int64(40))
		m2, _ := storage.NewMetric("Alloc", internal.GaugeTypeName, 13.345)
		m3, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(99))
		msgSlice := []message.Metrics{
			m1.GetMessageMetric(), m2.GetMessageMetric(), m3.GetMessageMetric(),
		}
		jsonMsg, _ := json.Marshal(msgSlice)

		url := "/updates/"
		b.ResetTimer()
		for i := 0; i < repeatBenchRun; i++ {
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))

			b.StopTimer()
			resetDBState(s.DBConn)
			b.StartTimer()
		}
	})
}
