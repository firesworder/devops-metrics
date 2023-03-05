package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"runtime"
	"time"
)

type gauge float64
type counter int64

var PollCount counter
var RandomValue gauge

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func main() {
	// первоначальное получение метрик
	memstats := runtime.MemStats{}
	runtime.ReadMemStats(&memstats)

	// подготовка тикеров на обновление и отправку
	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	for {
		select {
		case <-pollTicker.C:
			runtime.ReadMemStats(&memstats)
			PollCount++
			RandomValue = gauge(rand.Float64())
		case <-reportTicker.C:
			fmt.Println("Отправка метрик, с PollCount=", PollCount)
			sendMetric("Alloc", gauge(memstats.Alloc))
			sendMetric("BuckHashSys", gauge(memstats.BuckHashSys))
			sendMetric("Frees", gauge(memstats.Frees))

			sendMetric("GCCPUFraction", gauge(memstats.GCCPUFraction))
			sendMetric("GCSys", gauge(memstats.GCSys))
			sendMetric("HeapAlloc", gauge(memstats.HeapAlloc))

			sendMetric("HeapIdle", gauge(memstats.HeapIdle))
			sendMetric("HeapInuse", gauge(memstats.HeapInuse))
			sendMetric("HeapObjects", gauge(memstats.HeapObjects))

			sendMetric("HeapReleased", gauge(memstats.HeapReleased))
			sendMetric("HeapSys", gauge(memstats.HeapSys))
			sendMetric("LastGC", gauge(memstats.LastGC))

			sendMetric("Lookups", gauge(memstats.Lookups))
			sendMetric("MCacheInuse", gauge(memstats.MCacheInuse))
			sendMetric("MCacheSys", gauge(memstats.MCacheSys))

			sendMetric("MSpanInuse", gauge(memstats.MSpanInuse))
			sendMetric("MSpanSys", gauge(memstats.MSpanSys))
			sendMetric("Mallocs", gauge(memstats.Mallocs))

			sendMetric("NextGC", gauge(memstats.NextGC))
			sendMetric("NumForcedGC", gauge(memstats.NumForcedGC))
			sendMetric("NumGC", gauge(memstats.NumGC))

			sendMetric("OtherSys", gauge(memstats.OtherSys))
			sendMetric("PauseTotalNs", gauge(memstats.PauseTotalNs))
			sendMetric("StackInuse", gauge(memstats.StackInuse))

			sendMetric("StackSys", gauge(memstats.StackSys))
			sendMetric("Sys", gauge(memstats.Sys))
			sendMetric("TotalAlloc", gauge(memstats.TotalAlloc))

			// Кастомные метрики
			sendMetric("PollCount", counter(PollCount))
			sendMetric("RandomValue", gauge(RandomValue))
		}
	}
}

func sendMetric(paramName string, paramValue interface{}) {
	client := &http.Client{}
	var requestUrl string
	switch value := paramValue.(type) {
	case gauge:
		requestUrl = fmt.Sprintf("http://localhost:8080/update/%s/%s/%f", "gauge", paramName, value)
	case counter:
		requestUrl = fmt.Sprintf("http://localhost:8080/update/%s/%s/%d", "counter", paramName, value)
	default:
		panic("Незнакомый тип значения метрики")
	}
	request, err := http.NewRequest(http.MethodPost, requestUrl, nil)
	if err != nil {
		fmt.Println("Произошла ошибка при создании запроса:  ", err)
	}
	request.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Произошла ошибка при отправке запроса:", err)
	} else {
		fmt.Printf("Запрос отправлен с метрикой '%s', статус ответа %d\n", paramName, response.StatusCode)
	}
}
