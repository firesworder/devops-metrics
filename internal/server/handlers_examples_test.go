package server

import (
	"encoding/json"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"net/http"
	"net/http/httptest"
	"sort"
)

// Примеры

func ExampleServer_handlerShowAllMetrics() {
	// подготовка сервера для примера
	s, err := NewServer()
	if err != nil {
		panic(err)
	}
	s.LayoutsDir = "./html_layouts/"
	s.MetricStorage = storage.NewMemStorage(map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
		metric3.Name: *metric3,
	})
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	statusCode, contentType, content := sendRequest(http.MethodGet, ts.URL+"/", "text/plain", "")

	fmt.Println(statusCode)
	fmt.Println(contentType)
	fmt.Println(len(content) != 0)

	// Output:
	// 200
	// text/html; charset=utf-8
	// true
}

func ExampleServer_handlerGet() {
	s := getServer(false)
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	statusCode, contentType, content := sendRequest(
		http.MethodGet, ts.URL+"/value/counter/PollCount", "text/plain", "")

	fmt.Println(statusCode)
	fmt.Println(contentType)
	fmt.Println(content)

	// Output:
	// 200
	// text/plain; charset=utf-8
	// 10
}

func ExampleServer_handlerAddUpdateMetric() {
	s := getServer(false)
	nms := s.MetricStorage.(*storage.MemStorage)
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	urlParams := `/update/counter/PollCount/20`
	statusCode, _, _ := sendRequest(
		http.MethodPost, ts.URL+urlParams, "text/plain", "")

	fmt.Println(statusCode)
	fmt.Println(*exMetricCounter)
	fmt.Println(nms.Metrics[exMetricCounter.Name])

	// Output:
	// 200
	// {PollCount 10}
	// {PollCount 30}
}

func ExampleServer_handlerJSONAddUpdateMetric() {
	s := getServer(false)
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	metric, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(15))
	jsonMsg, _ := json.Marshal(metric.GetMessageMetric())
	url := "/update/"
	statusCode, contentType, content := sendRequest(
		http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))

	fmt.Println(statusCode)
	fmt.Println(contentType)
	fmt.Println(content)

	// Output:
	// 200
	// application/json
	// {"id":"CounterMetric1","type":"counter","delta":15}
}

func ExampleServer_handlerJSONGetMetric() {
	s := getServer(false)
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	jsonMsg, _ := json.Marshal(message.Metrics{ID: "PollCount"})
	url := "/value/"
	statusCode, contentType, content := sendRequest(
		http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))

	fmt.Println(statusCode)
	fmt.Println(contentType)
	fmt.Println(content)

	// Output:
	// 200
	// application/json
	// {"id":"PollCount","type":"counter","delta":10}
}

func ExampleServer_handlerPing() {
	// подготовка сервера для примера
	s, _ := NewServer()
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	urlParams := `/ping`
	statusCode, _, _ := sendRequest(
		http.MethodGet, ts.URL+urlParams, "", "")

	if Env.DatabaseDsn == "" {
		fmt.Println(statusCode == http.StatusInternalServerError)
	} else {
		fmt.Println(statusCode == http.StatusOK)
	}

	// Output:
	// true
}

func ExampleServer_handlerBatchUpdate() {
	s := getServer(false)
	nms := s.MetricStorage.(*storage.MemStorage)
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	m1, _ := storage.NewMetric("PollCount", internal.CounterTypeName, int64(40))
	m2, _ := storage.NewMetric("Alloc", internal.GaugeTypeName, 13.345)
	m3, _ := storage.NewMetric("CounterMetric1", internal.CounterTypeName, int64(99))
	msgSlice := []message.Metrics{
		m1.GetMessageMetric(), m2.GetMessageMetric(), m3.GetMessageMetric(),
	}
	jsonMsg, _ := json.Marshal(msgSlice)

	url := "/updates/"
	statusCode, _, _ := sendRequest(
		http.MethodPost, ts.URL+url, "application/json", string(jsonMsg))

	fmt.Println(statusCode)

	// упорядоченный (по названию метрики) вывод метрик
	var metricKeys []string
	for key := range nms.Metrics {
		metricKeys = append(metricKeys, key)
	}
	sort.Strings(metricKeys)
	for _, key := range metricKeys {
		fmt.Println(nms.Metrics[key])
	}
	// Output:
	// 200
	// {RandomValue 13.345}
	// {CounterMetric1 99}
	// {PollCount 50}
	// {RandomValue 12.133}
}
