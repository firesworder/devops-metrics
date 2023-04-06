package agent

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v7"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/go-resty/resty/v2"
	"log"
	"math/rand"
	"net/url"
	"runtime"
	"time"
)

type gauge float64
type counter int64

var memstats runtime.MemStats
var PollCount counter
var RandomValue gauge
var ServerURL string

type Environment struct {
	ServerAddress  string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
}

var Env Environment

func init() {
	InitCmdArgs()
	memstats = runtime.MemStats{}
	runtime.ReadMemStats(&memstats)
}

func InitServerURLByEnv() {
	ServerURL = (&url.URL{Scheme: "http", Host: Env.ServerAddress}).String()
}

// InitCmdArgs Определяет флаги командной строки и линкует их с соотв полями объекта Env
// В рамках этой же функции происходит и заполнение дефолтными значениями
func InitCmdArgs() {
	flag.StringVar(&Env.ServerAddress, "a", "localhost:8080", "Server address")
	flag.DurationVar(&Env.ReportInterval, "r", 10*time.Second, "report interval")
	flag.DurationVar(&Env.PollInterval, "p", 2*time.Second, "poll(update) interval")
}

// ParseEnvArgs Парсит значения полей Env. Сначала из cmd аргументов, затем из перем-х окружения
func ParseEnvArgs() {
	// Парсинг аргументов cmd
	flag.Parse()

	// Парсинг перем окружения
	err := env.Parse(&Env)
	if err != nil {
		panic(err)
	}
}

func UpdateMetrics() {
	runtime.ReadMemStats(&memstats)
	PollCount++
	RandomValue = gauge(rand.Float64())
}

func SendMetrics() {
	sendMetricByJson("Alloc", gauge(memstats.Alloc))
	sendMetricByJson("BuckHashSys", gauge(memstats.BuckHashSys))
	sendMetricByJson("Frees", gauge(memstats.Frees))

	sendMetricByJson("GCCPUFraction", gauge(memstats.GCCPUFraction))
	sendMetricByJson("GCSys", gauge(memstats.GCSys))
	sendMetricByJson("HeapAlloc", gauge(memstats.HeapAlloc))

	sendMetricByJson("HeapIdle", gauge(memstats.HeapIdle))
	sendMetricByJson("HeapInuse", gauge(memstats.HeapInuse))
	sendMetricByJson("HeapObjects", gauge(memstats.HeapObjects))

	sendMetricByJson("HeapReleased", gauge(memstats.HeapReleased))
	sendMetricByJson("HeapSys", gauge(memstats.HeapSys))
	sendMetricByJson("LastGC", gauge(memstats.LastGC))

	sendMetricByJson("Lookups", gauge(memstats.Lookups))
	sendMetricByJson("MCacheInuse", gauge(memstats.MCacheInuse))
	sendMetricByJson("MCacheSys", gauge(memstats.MCacheSys))

	sendMetricByJson("MSpanInuse", gauge(memstats.MSpanInuse))
	sendMetricByJson("MSpanSys", gauge(memstats.MSpanSys))
	sendMetricByJson("Mallocs", gauge(memstats.Mallocs))

	sendMetricByJson("NextGC", gauge(memstats.NextGC))
	sendMetricByJson("NumForcedGC", gauge(memstats.NumForcedGC))
	sendMetricByJson("NumGC", gauge(memstats.NumGC))

	sendMetricByJson("OtherSys", gauge(memstats.OtherSys))
	sendMetricByJson("PauseTotalNs", gauge(memstats.PauseTotalNs))
	sendMetricByJson("StackInuse", gauge(memstats.StackInuse))

	sendMetricByJson("StackSys", gauge(memstats.StackSys))
	sendMetricByJson("Sys", gauge(memstats.Sys))
	sendMetricByJson("TotalAlloc", gauge(memstats.TotalAlloc))

	// Кастомные метрики
	sendMetricByJson("PollCount", counter(PollCount))
	sendMetricByJson("RandomValue", gauge(RandomValue))
}

// sendMetricByJson Отправляет метрику Post запросом, посредством url.
// Пока что не обрабатывает ответ сервера, ошибки выбрасывает в консоль!
func sendMetricByURL(paramName string, paramValue interface{}) {
	client := resty.New()
	var requestURL string
	switch value := paramValue.(type) {
	case gauge:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%f", ServerURL, "gauge", paramName, value)
	case counter:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%d", ServerURL, "counter", paramName, value)
	default:
		log.Printf("unhandled metric type '%T'", value)
	}

	_, err := client.R().
		SetHeader("Content-Type", "text/plain").
		Post(requestURL)
	if err != nil {
		log.Println(err)
	}
}

// sendMetricByJson Отправляет метрику Post запросом, в Json формате.
// Пока что не обрабатывает ответ сервера, ошибки выбрасывает в консоль!
func sendMetricByJson(paramName string, paramValue interface{}) {
	client := resty.New()
	client.SetBaseURL(ServerURL)
	var msg message.Metrics
	msg.ID = paramName
	switch value := paramValue.(type) {
	case gauge:
		msg.MType = internal.GaugeTypeName
		float64Val := float64(value)
		msg.Value = &float64Val
	case counter:
		msg.MType = internal.CounterTypeName
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

	_, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(jsonBody).
		Post(`/update/`)
	if err != nil {
		log.Println(err)
		return
	}
}
