package agent

import (
	"encoding/json"
	"flag"
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

// todo: реализовать две версии отправки сообщений.
//  одна через json, вторая - старая, через url. Можно реализовать sendMetricJson и вызывать нужный потом

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
