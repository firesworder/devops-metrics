package agent

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v7"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"golang.org/x/sync/errgroup"
	"log"
	"math/rand"
	"net/url"
	"runtime"
	"sync"
	"time"
)

type gauge float64
type counter int64

var memstats runtime.MemStats
var PollCount counter
var RandomValue gauge
var ServerURL string
var goPsutilStats GoPsutilStats

var updateMetricsMutex sync.RWMutex

// bool значение в канале как метка, что нужно выполнить задачу
var sendWorkPoolCh chan bool

// Для тестирования функции UpdateMetrics
var testUMWG *sync.WaitGroup

type GoPsutilStats struct {
	TotalMemory    float64
	FreeMemory     float64
	CPUutilization []float64
}

type Environment struct {
	ServerAddress  string        `env:"ADDRESS"`
	ReportInterval time.Duration `env:"REPORT_INTERVAL"`
	PollInterval   time.Duration `env:"POLL_INTERVAL"`
	Key            string        `env:"KEY"`
	RateLimit      int           `env:"RATE_LIMIT"`
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
	flag.IntVar(&Env.RateLimit, "l", 0, "rate limit(send routines at one time)")
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

func InitWorkPool() {
	workersCount := Env.RateLimit
	if Env.RateLimit == 0 {
		workersCount = 1
	}

	sendWorkPoolCh = make(chan bool, workersCount)

	// создать и запустить воркеры
	wg := &sync.WaitGroup{}
	wg.Add(workersCount)
	for i := 0; i < workersCount; i++ {
		go func(workerIndex int, wg *sync.WaitGroup) {
			wg.Done() // сигнал о том, что горутина-воркер запустилась
			for _ = range sendWorkPoolCh {
				log.Printf("worker with index '%d' used for SendMetrics()", workerIndex)
				SendMetrics()
			}
		}(i, wg)
	}
	wg.Wait()
}

func FinishWorkPool() {
	close(sendWorkPoolCh)
}

func UpdateMetrics() {
	// полностью блокируем данные метрик на время обновления
	updateMetricsMutex.Lock()
	defer updateMetricsMutex.Unlock()
	defer func() {
		if testUMWG != nil {
			testUMWG.Done()
		}
	}()

	g, _ := errgroup.WithContext(context.Background())
	g.Go(updateMemStats)
	g.Go(updateGoPsutilStats)

	if err := g.Wait(); err != nil {
		log.Println(err)
		return
	}
}

func updateMemStats() (err error) {
	runtime.ReadMemStats(&memstats)
	PollCount++
	RandomValue = gauge(rand.Float64())
	return
}

func updateGoPsutilStats() (err error) {
	vM, err := mem.VirtualMemory()
	if err != nil {
		return
	}
	goPsutilStats.TotalMemory = float64(vM.Total)
	goPsutilStats.FreeMemory = float64(vM.Free)

	cpuS, err := cpu.Percent(500*time.Millisecond, true)
	if err != nil {
		return
	}
	goPsutilStats.CPUutilization = cpuS
	return
}

func CreateSendMetricsJob(ctx context.Context) {
	select {
	case sendWorkPoolCh <- true:
		log.Println("job sent into workpool channel")
	case <-ctx.Done():
		log.Println("job was canceled by context")
	}
}

func SendMetrics() {
	updateMetricsMutex.RLock()
	defer updateMetricsMutex.RUnlock()

	metrics := map[string]interface{}{
		"Alloc":       gauge(memstats.Alloc),
		"BuckHashSys": gauge(memstats.BuckHashSys),
		"Frees":       gauge(memstats.Frees),

		"GCCPUFraction": gauge(memstats.GCCPUFraction),
		"GCSys":         gauge(memstats.GCSys),
		"HeapAlloc":     gauge(memstats.HeapAlloc),

		"HeapIdle":    gauge(memstats.HeapIdle),
		"HeapInuse":   gauge(memstats.HeapInuse),
		"HeapObjects": gauge(memstats.HeapObjects),

		"HeapReleased": gauge(memstats.HeapReleased),
		"HeapSys":      gauge(memstats.HeapSys),
		"LastGC":       gauge(memstats.LastGC),

		"Lookups":     gauge(memstats.Lookups),
		"MCacheInuse": gauge(memstats.MCacheInuse),
		"MCacheSys":   gauge(memstats.MCacheSys),

		"MSpanInuse": gauge(memstats.MSpanInuse),
		"MSpanSys":   gauge(memstats.MSpanSys),
		"Mallocs":    gauge(memstats.Mallocs),

		"NextGC":      gauge(memstats.NextGC),
		"NumForcedGC": gauge(memstats.NumForcedGC),
		"NumGC":       gauge(memstats.NumGC),

		"OtherSys":     gauge(memstats.OtherSys),
		"PauseTotalNs": gauge(memstats.PauseTotalNs),
		"StackInuse":   gauge(memstats.StackInuse),

		"StackSys":   gauge(memstats.StackSys),
		"Sys":        gauge(memstats.Sys),
		"TotalAlloc": gauge(memstats.TotalAlloc),

		// Кастомные метрики
		"PollCount":   counter(PollCount),
		"RandomValue": gauge(RandomValue),
	}

	// goPSUtil метрики
	metrics["TotalMemory"] = gauge(goPsutilStats.TotalMemory)
	metrics["FreeMemory"] = gauge(goPsutilStats.FreeMemory)
	var metricID string
	for i, cpuUtilStat := range goPsutilStats.CPUutilization {
		metricID = fmt.Sprintf("CPUutilization%d", i)
		metrics[metricID] = gauge(cpuUtilStat)
	}

	sendMetricsBatchByJSON(metrics)
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
		Post(`/updates/`)
	if err != nil {
		log.Println(err)
		return
	}
}
