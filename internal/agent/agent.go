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
	Key            string        `env:"KEY"`
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
	flag.StringVar(&Env.Key, "k", "", "key for hash func")
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
	sendMetricByJSON("Alloc", gauge(memstats.Alloc))
	sendMetricByJSON("BuckHashSys", gauge(memstats.BuckHashSys))
	sendMetricByJSON("Frees", gauge(memstats.Frees))

	sendMetricByJSON("GCCPUFraction", gauge(memstats.GCCPUFraction))
	sendMetricByJSON("GCSys", gauge(memstats.GCSys))
	sendMetricByJSON("HeapAlloc", gauge(memstats.HeapAlloc))

	sendMetricByJSON("HeapIdle", gauge(memstats.HeapIdle))
	sendMetricByJSON("HeapInuse", gauge(memstats.HeapInuse))
	sendMetricByJSON("HeapObjects", gauge(memstats.HeapObjects))

	sendMetricByJSON("HeapReleased", gauge(memstats.HeapReleased))
	sendMetricByJSON("HeapSys", gauge(memstats.HeapSys))
	sendMetricByJSON("LastGC", gauge(memstats.LastGC))

	sendMetricByJSON("Lookups", gauge(memstats.Lookups))
	sendMetricByJSON("MCacheInuse", gauge(memstats.MCacheInuse))
	sendMetricByJSON("MCacheSys", gauge(memstats.MCacheSys))

	sendMetricByJSON("MSpanInuse", gauge(memstats.MSpanInuse))
	sendMetricByJSON("MSpanSys", gauge(memstats.MSpanSys))
	sendMetricByJSON("Mallocs", gauge(memstats.Mallocs))

	sendMetricByJSON("NextGC", gauge(memstats.NextGC))
	sendMetricByJSON("NumForcedGC", gauge(memstats.NumForcedGC))
	sendMetricByJSON("NumGC", gauge(memstats.NumGC))

	sendMetricByJSON("OtherSys", gauge(memstats.OtherSys))
	sendMetricByJSON("PauseTotalNs", gauge(memstats.PauseTotalNs))
	sendMetricByJSON("StackInuse", gauge(memstats.StackInuse))

	sendMetricByJSON("StackSys", gauge(memstats.StackSys))
	sendMetricByJSON("Sys", gauge(memstats.Sys))
	sendMetricByJSON("TotalAlloc", gauge(memstats.TotalAlloc))

	// Кастомные метрики
	sendMetricByJSON("PollCount", counter(PollCount))
	sendMetricByJSON("RandomValue", gauge(RandomValue))
}

// sendMetricByURL Отправляет метрику Post запросом, посредством url.
// Пока что не обрабатывает ответ сервера, ошибки выбрасывает в консоль!
func sendMetricByURL(paramName string, paramValue interface{}) {
	client := resty.New()
	var requestURL string
	switch value := paramValue.(type) {
	case gauge:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%f", ServerURL, internal.GaugeTypeName, paramName, value)
	case counter:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%d", ServerURL, internal.CounterTypeName, paramName, value)
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

// sendMetricByJSON Отправляет метрику Post запросом, в Json формате.
// Пока что не обрабатывает ответ сервера, ошибки выбрасывает в консоль!
func sendMetricByJSON(paramName string, paramValue interface{}) {
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

	if Env.Key != "" {
		err := msg.InitHash(Env.Key)
		if err != nil {
			log.Println(err)
			return
		}
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

func sendMetricsBatchByJSON(metrics map[string]interface{}) {
	client := resty.New()
	client.SetBaseURL(ServerURL)

	var metricsToSend []message.Metrics
	var msg *message.Metrics
	for mN, mV := range metrics {
		msg = &message.Metrics{}

		msg.ID = mN
		switch value := mV.(type) {
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

		if Env.Key != "" {
			err := msg.InitHash(Env.Key)
			if err != nil {
				log.Println(err)
				return
			}
		}

		metricsToSend = append(metricsToSend, *msg)
	}

	jsonBody, err := json.Marshal(metricsToSend)
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
