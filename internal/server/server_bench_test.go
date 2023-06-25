package server

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

var (
	benchMetricCounter, _ = storage.NewMetric("PollCount", internal.CounterTypeName, int64(10))
	benchMetricGauge1, _  = storage.NewMetric("RandomValue", internal.GaugeTypeName, 12.133)
	benchMetricGauge2, _  = storage.NewMetric("Alloc", internal.GaugeTypeName, 7.77)
)

func getMetricsMap() map[string]storage.Metric {
	return map[string]storage.Metric{
		benchMetricCounter.Name: *benchMetricCounter,
		benchMetricGauge1.Name:  *benchMetricGauge1,
		benchMetricGauge2.Name:  *benchMetricGauge2,
	}
}

func getServer() (*Server, *storage.MemStorage) {
	// подготовка сервера для примера
	s, err := NewServer()
	if err != nil {
		panic(err)
	}
	s.LayoutsDir = "./html_layouts/"
	nms := storage.NewMemStorage(getMetricsMap())
	s.MetricStorage = nms
	return s, nms
}

func sendRequest(method, url, contentType, content string) (int, string, string) {
	// создаю реквест
	req, _ := http.NewRequest(method, url, strings.NewReader(content))
	req.Header.Set("Content-Type", contentType)

	// делаю реквест на дефолтном клиенте
	resp, _ := http.DefaultClient.Do(req)

	// читаю ответ сервера
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, resp.Header.Get("Content-Type"), string(respBody)
}

func BenchmarkHandlers(b *testing.B) {
	b.Run("getMetric", func(b *testing.B) {
		s, _ := getServer()
		ts := httptest.NewServer(s.Router)
		defer ts.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sendRequest(http.MethodGet, ts.URL+"/value/counter/PollCount", "text/plain", "")
		}
	})

	b.Run("showAllMetrics", func(b *testing.B) {
		s, _ := getServer()
		ts := httptest.NewServer(s.Router)
		defer ts.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sendRequest(http.MethodGet, ts.URL+"/", "text/plain", "")
		}
	})

	b.Run("AddUpdateMetric", func(b *testing.B) {
		s, nms := getServer()
		ts := httptest.NewServer(s.Router)
		defer ts.Close()

		urlParams := `/update/counter/PollCount/20`
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			nms.Metrics = getMetricsMap()
			b.StartTimer()
			sendRequest(http.MethodPost, ts.URL+urlParams, "text/plain", "")
		}
	})

	b.Run("JSONAddUpdateMetric", func(b *testing.B) {
		s, nms := getServer()
		ts := httptest.NewServer(s.Router)
		defer ts.Close()

		metric, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(15))
		jsonMsg, _ := json.Marshal(metric.GetMessageMetric())
		url := "/update/"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			nms.Metrics = getMetricsMap()
			b.StartTimer()
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))
		}
	})

	b.Run("JSONGetMetric", func(b *testing.B) {
		s, _ := getServer()
		ts := httptest.NewServer(s.Router)
		defer ts.Close()

		jsonMsg, _ := json.Marshal(message.Metrics{ID: "PollCount"})
		url := "/value/"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))
		}
	})

	b.Run("BatchUpdate", func(b *testing.B) {
		s, nms := getServer()
		ts := httptest.NewServer(s.Router)
		defer ts.Close()

		m1, _ := storage.NewMetric("PollCount", internal.CounterTypeName, int64(40))
		m2, _ := storage.NewMetric("Alloc", internal.GaugeTypeName, 13.345)
		m3, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(99))
		msgSlice := []message.Metrics{
			m1.GetMessageMetric(), m2.GetMessageMetric(), m3.GetMessageMetric(),
		}
		jsonMsg, _ := json.Marshal(msgSlice)

		url := "/updates/"
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			nms.Metrics = getMetricsMap()
			b.StartTimer()
			sendRequest(http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))
		}
	})
}
