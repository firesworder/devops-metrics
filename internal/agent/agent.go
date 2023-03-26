package agent

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/go-resty/resty/v2"
	"log"
	"math/rand"
	"net/url"
	"runtime"
)

type gauge float64
type counter int64

var memstats runtime.MemStats
var PollCount counter
var RandomValue gauge

// todo: реализовать две версии отправки сообщений.
//  одна через json, вторая - старая, через url. Можно реализовать sendMetricJson и вызывать нужный потом

var ServerURL = `http://localhost:8080`
var AddUpdateMetricUrl = `/update`

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

// todo добавить возвр. ответа + ошибки
func sendMetric(paramName string, paramValue interface{}) {
	client := resty.New()
	var msg message.Metrics
	msg.ID = paramName
	switch value := paramValue.(type) {
	case gauge:
		msg.MType = "gauge"
		float64Val := float64(value)
		msg.Value = &float64Val
	case counter:
		msg.MType = "counter"
		int64Val := int64(value)
		msg.Delta = &int64Val
	default:
		log.Printf("unhandled metric type '%T'", value)
		return
	}

	jsonBody, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}

	reqUrl, err := url.JoinPath(ServerURL, AddUpdateMetricUrl)
	if err != nil {
		log.Println(err)
		return
	}

	_, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		Post(reqUrl)
	if err != nil {
		log.Println(err)
		return
	}
}
