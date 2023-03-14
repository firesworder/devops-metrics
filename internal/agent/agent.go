package agent

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
)

type gauge float64
type counter int64

var memstats runtime.MemStats
var PollCount counter
var RandomValue gauge

var ServerURL = `http://localhost:8080`

func init() {
	memstats = runtime.MemStats{}
	runtime.ReadMemStats(&memstats)
}

func UpdateMetrics() {
	runtime.ReadMemStats(&memstats)
	PollCount++
	RandomValue = gauge(rand.Float64())
}

// todo: хорошо бы покрыть тестами и переписать на фреймворк(хотя не понятно, что тестировать)
func SendMetrics() {
	_ = sendMetric("Alloc", gauge(memstats.Alloc))
	_ = sendMetric("BuckHashSys", gauge(memstats.BuckHashSys))
	_ = sendMetric("Frees", gauge(memstats.Frees))

	_ = sendMetric("GCCPUFraction", gauge(memstats.GCCPUFraction))
	_ = sendMetric("GCSys", gauge(memstats.GCSys))
	_ = sendMetric("HeapAlloc", gauge(memstats.HeapAlloc))

	_ = sendMetric("HeapIdle", gauge(memstats.HeapIdle))
	_ = sendMetric("HeapInuse", gauge(memstats.HeapInuse))
	_ = sendMetric("HeapObjects", gauge(memstats.HeapObjects))

	_ = sendMetric("HeapReleased", gauge(memstats.HeapReleased))
	_ = sendMetric("HeapSys", gauge(memstats.HeapSys))
	_ = sendMetric("LastGC", gauge(memstats.LastGC))

	_ = sendMetric("Lookups", gauge(memstats.Lookups))
	_ = sendMetric("MCacheInuse", gauge(memstats.MCacheInuse))
	_ = sendMetric("MCacheSys", gauge(memstats.MCacheSys))

	_ = sendMetric("MSpanInuse", gauge(memstats.MSpanInuse))
	_ = sendMetric("MSpanSys", gauge(memstats.MSpanSys))
	_ = sendMetric("Mallocs", gauge(memstats.Mallocs))

	_ = sendMetric("NextGC", gauge(memstats.NextGC))
	_ = sendMetric("NumForcedGC", gauge(memstats.NumForcedGC))
	_ = sendMetric("NumGC", gauge(memstats.NumGC))

	_ = sendMetric("OtherSys", gauge(memstats.OtherSys))
	_ = sendMetric("PauseTotalNs", gauge(memstats.PauseTotalNs))
	_ = sendMetric("StackInuse", gauge(memstats.StackInuse))

	_ = sendMetric("StackSys", gauge(memstats.StackSys))
	_ = sendMetric("Sys", gauge(memstats.Sys))
	_ = sendMetric("TotalAlloc", gauge(memstats.TotalAlloc))

	// Кастомные метрики
	_ = sendMetric("PollCount", counter(PollCount))
	_ = sendMetric("RandomValue", gauge(RandomValue))
}

func sendMetric(paramName string, paramValue interface{}) (err error) {
	client := &http.Client{}
	var requestURL string
	// todo: переписать как один спринтф, передавая тип строкой
	switch value := paramValue.(type) {
	case gauge:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%f", ServerURL, "gauge", paramName, value)
	case counter:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%d", ServerURL, "counter", paramName, value)
	default:
		return fmt.Errorf("unhandled metric type '%T'", value)
	}

	request, err := http.NewRequest(http.MethodPost, requestURL, nil)
	if err != nil {
		return err
	}

	request.Header.Add("Content-Type", "text/plain")
	response, err := client.Do(request)
	if err != nil {
		return err
	}

	// закрываю тело ответа
	defer response.Body.Close()
	_, err = io.ReadAll(response.Body)
	if err != nil {
		return err
	}
	return nil
}
