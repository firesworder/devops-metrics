package main

import (
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/server"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"net/http"
)

func main() {
	//metricHandler := server.NewDefaultMetricHandler()
	serverParams := server.NewServer()
	metric1, _ := storage.NewMetric("PollCount", "counter", int64(10))
	metric2, _ := storage.NewMetric("RandomValue", "gauge", 12.133)
	metric3, _ := storage.NewMetric("Alloc", "gauge", 7.77)
	serverParams.MetricStorage.AddMetric(*metric1)
	serverParams.MetricStorage.AddMetric(*metric2)
	serverParams.MetricStorage.AddMetric(*metric3)
	serverObj := &http.Server{
		Addr:    "localhost:8080",
		Handler: serverParams.Router,
	}
	err := serverObj.ListenAndServe()
	if err != nil {
		fmt.Println("Произошла ошибка при запуске сервера:", err)
		return
	}
}
