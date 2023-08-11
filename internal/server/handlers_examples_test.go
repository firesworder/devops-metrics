package server

import (
	"encoding/json"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
)

func ExampleServer_handlerShowAllMetrics() {
	server, err := NewTempServer()
	if err != nil {
		log.Fatal(err)
	}

	s := HTTPServer{server: server}
	// getMetricsMap возвращает словарь метрик(map[string]storage.Metric) для демонстрации
	s.server.MetricStorage = storage.NewMemStorage(getMetricsMap())
	s.server.LayoutsDir = "./html_layouts/"
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
	server, err := NewTempServer()
	if err != nil {
		log.Fatal(err)
	}

	s := HTTPServer{server: server}
	// getMetricsMap возвращает словарь метрик(map[string]storage.Metric) для демонстрации
	s.server.MetricStorage = storage.NewMemStorage(getMetricsMap())
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
	server, err := NewTempServer()
	if err != nil {
		log.Fatal(err)
	}

	s := HTTPServer{server: server}
	// getMetricsMap возвращает словарь метрик(map[string]storage.Metric) для демонстрации
	s.server.MetricStorage = storage.NewMemStorage(getMetricsMap())
	nms := s.server.MetricStorage.(*storage.MemStorage)
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
	// {10 PollCount}
	// {30 PollCount}
}

func ExampleServer_handlerJSONAddUpdateMetric() {
	server, err := NewTempServer()
	if err != nil {
		log.Fatal(err)
	}

	s := HTTPServer{server: server}
	// getMetricsMap возвращает словарь метрик(map[string]storage.Metric) для демонстрации
	s.server.MetricStorage = storage.NewMemStorage(getMetricsMap())
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
	server, err := NewTempServer()
	if err != nil {
		log.Fatal(err)
	}

	s := HTTPServer{server: server}
	// getMetricsMap возвращает словарь метрик(map[string]storage.Metric) для демонстрации
	s.server.MetricStorage = storage.NewMemStorage(getMetricsMap())
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
	server, err := NewTempServer()
	if err != nil {
		log.Fatal(err)
	}

	s := HTTPServer{server: server}
	// getMetricsMap возвращает словарь метрик(map[string]storage.Metric) для демонстрации
	s.server.MetricStorage = storage.NewMemStorage(getMetricsMap())
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
	server, err := NewTempServer()
	if err != nil {
		log.Fatal(err)
	}

	s := HTTPServer{server: server}
	// getMetricsMap возвращает словарь метрик(map[string]storage.Metric) для демонстрации
	s.server.MetricStorage = storage.NewMemStorage(getMetricsMap())
	nms := s.server.MetricStorage.(*storage.MemStorage)
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
	// {13.345 Alloc}
	// {99 CounterMetric1}
	// {50 PollCount}
	// {12.133 RandomValue}
}
