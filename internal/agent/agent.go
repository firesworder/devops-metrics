package agent

import (
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"runtime"
)

// todo: переписать на использование фреймворка

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

func SendMetrics() {
	errorByMetric := make(map[string]error)
	errorByMetric["Alloc"] = sendMetric("Alloc", gauge(memstats.Alloc))
	errorByMetric["BuckHashSys"] = sendMetric("BuckHashSys", gauge(memstats.BuckHashSys))
	errorByMetric["Frees"] = sendMetric("Frees", gauge(memstats.Frees))

	errorByMetric["GCCPUFraction"] = sendMetric("GCCPUFraction", gauge(memstats.GCCPUFraction))
	errorByMetric["GCSys"] = sendMetric("GCSys", gauge(memstats.GCSys))
	errorByMetric["HeapAlloc"] = sendMetric("HeapAlloc", gauge(memstats.HeapAlloc))

	errorByMetric["HeapIdle"] = sendMetric("HeapIdle", gauge(memstats.HeapIdle))
	errorByMetric["HeapInuse"] = sendMetric("HeapInuse", gauge(memstats.HeapInuse))
	errorByMetric["HeapObjects"] = sendMetric("HeapObjects", gauge(memstats.HeapObjects))

	errorByMetric["HeapReleased"] = sendMetric("HeapReleased", gauge(memstats.HeapReleased))
	errorByMetric["HeapSys"] = sendMetric("HeapSys", gauge(memstats.HeapSys))
	errorByMetric["LastGC"] = sendMetric("LastGC", gauge(memstats.LastGC))

	errorByMetric["Lookups"] = sendMetric("Lookups", gauge(memstats.Lookups))
	errorByMetric["MCacheInuse"] = sendMetric("MCacheInuse", gauge(memstats.MCacheInuse))
	errorByMetric["MCacheSys"] = sendMetric("MCacheSys", gauge(memstats.MCacheSys))

	errorByMetric["MSpanInuse"] = sendMetric("MSpanInuse", gauge(memstats.MSpanInuse))
	errorByMetric["MSpanSys"] = sendMetric("MSpanSys", gauge(memstats.MSpanSys))
	errorByMetric["Mallocs"] = sendMetric("Mallocs", gauge(memstats.Mallocs))

	errorByMetric["NextGC"] = sendMetric("NextGC", gauge(memstats.NextGC))
	errorByMetric["NumForcedGC"] = sendMetric("NumForcedGC", gauge(memstats.NumForcedGC))
	errorByMetric["NumGC"] = sendMetric("NumGC", gauge(memstats.NumGC))

	errorByMetric["OtherSys"] = sendMetric("OtherSys", gauge(memstats.OtherSys))
	errorByMetric["PauseTotalNs"] = sendMetric("PauseTotalNs", gauge(memstats.PauseTotalNs))
	errorByMetric["StackInuse"] = sendMetric("StackInuse", gauge(memstats.StackInuse))

	errorByMetric["StackSys"] = sendMetric("StackSys", gauge(memstats.StackSys))
	errorByMetric["Sys"] = sendMetric("Sys", gauge(memstats.Sys))
	errorByMetric["TotalAlloc"] = sendMetric("TotalAlloc", gauge(memstats.TotalAlloc))

	// Кастомные метрики
	errorByMetric["PollCount"] = sendMetric("PollCount", counter(PollCount))
	errorByMetric["RandomValue"] = sendMetric("RandomValue", gauge(RandomValue))

	checkMetricSendingErrors(errorByMetric, PollCount, RandomValue)
}

func checkMetricSendingErrors(errorsMap map[string]error, PollCount counter, RandomValue gauge) {
	var errorsCount counter
	for metricName := range errorsMap {
		if errorsMap[metricName] != nil {
			errorsCount++
			fmt.Printf("Metric '%s' has error: %s\n", metricName, errorsMap[metricName])
		}
	}
	if errorsCount > 0 {
		fmt.Printf("Found %d errors, for PollCount: %d and RandomValue: %f", errorsCount, PollCount, RandomValue)
	}
}

func sendMetric(paramName string, paramValue interface{}) (err error) {
	client := &http.Client{}
	var requestURL string
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
