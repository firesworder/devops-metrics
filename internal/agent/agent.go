// Package agent реализует работу агента сбора метрик(в cmd/agent/ используются функции этого пакета).
// Агент собирает требуемые(по заданию) метрики, подготавливает их к отправке и отправляет на сервер.
// Также в рамках этого пакета определены функции работающие с переменными окружения агента.
package agent

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/crypt"
	"log"
	"math/rand"
	"net/url"
	"runtime"
	"sync"
	"time"

	"github.com/caarlos0/env/v7"
	"github.com/go-resty/resty/v2"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"golang.org/x/sync/errgroup"

	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
)

// типы необходимые для использования с метриками(по заданию).
type (
	gauge   float64
	counter int64
)

// serverURL содержит адрес сервера.
var serverURL string

// переменные в которых хранятся значения метрик(в сыром виде) для отправки.
var (
	memstats      runtime.MemStats
	pollCount     counter
	randomValue   gauge
	goPsutilStats = struct {
		CPUutilization []float64
		TotalMemory    float64
		FreeMemory     float64
	}{}
)

// updateMetricsMutex для RW блокировки переменных со значениями метрик на время записи и чтения.
var updateMetricsMutex sync.RWMutex

// Для тестирования функции UpdateMetrics
var testUMWG *sync.WaitGroup

// environment для получения(из ENV и cmd) и хранения переменных окружения агента.
type environment struct {
	Key               string        `env:"KEY"`
	ServerAddress     string        `env:"ADDRESS"`
	RateLimit         int           `env:"RATE_LIMIT"`
	ReportInterval    time.Duration `env:"REPORT_INTERVAL"`
	PollInterval      time.Duration `env:"POLL_INTERVAL"`
	PublicCryptoKeyFp string        `env:"CRYPTO_KEY"`
	ConfigFilepath    string        `env:"CONFIG"`
}

// workPool содержит переменные служебного использования для воркпула.
type workPool struct {
	ch           chan bool
	workersCount int
	wgStart      sync.WaitGroup
	wgFinish     sync.WaitGroup
}

// WPool воркпул, отправляющий метрики на сервер.
var WPool workPool

// Env объект с переменными окружения(из ENV и cmd args).
var Env environment

var encoder *crypt.Encoder

func init() {
	InitCmdArgs()
	if Env.PublicCryptoKeyFp != "" {
		var err error
		encoder, err = crypt.NewEncoder(Env.PublicCryptoKeyFp)
		if err != nil {
			panic(err)
		}
	}
	memstats = runtime.MemStats{}
	runtime.ReadMemStats(&memstats)
}

// InitServerURLByEnv Устанавливает глоб-ую переменную serverURL по переменной окружения ServerAddress.
func InitServerURLByEnv() {
	serverURL = (&url.URL{Scheme: "http", Host: Env.ServerAddress}).String()
}

// InitCmdArgs Определяет флаги командной строки и линкует их с соотв полями объекта Env.
// В рамках этой же функции происходит и заполнение дефолтными значениями.
func InitCmdArgs() {
	flag.StringVar(&Env.ServerAddress, "a", "localhost:8080", "Server address")
	flag.DurationVar(&Env.ReportInterval, "r", 10*time.Second, "report interval")
	flag.DurationVar(&Env.PollInterval, "p", 2*time.Second, "poll(update) interval")
	flag.StringVar(&Env.Key, "k", "", "key for hash func")
	flag.IntVar(&Env.RateLimit, "l", 0, "rate limit(send routines at one time)")
	flag.StringVar(&Env.PublicCryptoKeyFp, "crypto-key", "", "filepath to public key")
	flag.StringVar(&Env.ConfigFilepath, "config", "", "filepath to json env config")
	flag.StringVar(&Env.ConfigFilepath, "c", "", "filepath to json env config")
}

// ParseEnvArgs Парсит значения полей Env. Сначала из cmd аргументов, затем из перем-х окружения.
func ParseEnvArgs() {
	// Парсинг аргументов cmd
	flag.Parse()

	// Парсинг перем окружения
	err := env.Parse(&Env)
	if err != nil {
		panic(err)
	}

	if Env.ConfigFilepath != "" {
		err = parseJSONConfig()
		if err != nil {
			panic(err)
		}
	}
}

// Start запускает воркпул. Возвращает управление когда все воркеры запущены.
func (wp *workPool) Start() {
	// определен. кол-ва воркеров
	if Env.RateLimit == 0 {
		wp.workersCount = 1
	} else {
		wp.workersCount = Env.RateLimit
	}

	// определение буф.канала
	wp.ch = make(chan bool, wp.workersCount)

	// определение waitGroup для запуска и завершения воркеров
	wp.wgStart, wp.wgFinish = sync.WaitGroup{}, sync.WaitGroup{}
	wp.wgStart.Add(wp.workersCount)
	wp.wgFinish.Add(wp.workersCount)

	// создание и запуск воркеров
	for i := 0; i < wp.workersCount; i++ {
		go func(workerIndex int) {
			wp.wgStart.Done() // сигнал о том, что горутина-воркер запустилась
			for range wp.ch {
				log.Printf("worker with index '%d' used for sendMetrics()", workerIndex)
				sendMetrics()
			}
			wp.wgFinish.Done()
		}(i)
	}
	// ждать пока все воркеры запустятся
	wp.wgStart.Wait()
}

// Close останавливает воркпул. Также как и Start дожидается завершения всех воркеров.
func (wp *workPool) Close() {
	close(wp.ch)
	wp.wgFinish.Wait()
}

// UpdateMetrics основная(вызываемая в cmd) функция получения метрик для отправки.
// Внутри отдельными горутинами собираются данные из memstats и go-psutil.
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

// updateMemStats получает актуальные значения метрик из memstats.
func updateMemStats() (err error) {
	runtime.ReadMemStats(&memstats)
	pollCount++
	randomValue = gauge(rand.Float64())
	return
}

// updateGoPsutilStats получает актуальные значения метрик из go-psutil.
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

// CreateSendMetricsJob создает задание на отправку метрик в воркпуле.
func (wp *workPool) CreateSendMetricsJob(ctx context.Context) {
	select {
	case wp.ch <- true:
		log.Println("job sent into workpool channel")
	case <-ctx.Done():
		log.Println("job was canceled by context")
	}
}

// sendMetrics отправляет метрики на сервер.
func sendMetrics() {
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
		"PollCount":   counter(pollCount),
		"RandomValue": gauge(randomValue),
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

// sendMetricByURL отправляет метрику Post запросом, посредством url.
func sendMetricByURL(paramName string, paramValue interface{}) {
	client := resty.New()
	var requestURL string
	switch value := paramValue.(type) {
	case gauge:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%f", serverURL, internal.GaugeTypeName, paramName, value)
	case counter:
		requestURL = fmt.Sprintf("%s/update/%s/%s/%d", serverURL, internal.CounterTypeName, paramName, value)
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

// sendMetricByJSON отправляет метрику Post запросом, в Json формате.
func sendMetricByJSON(paramName string, paramValue interface{}) {
	var err error

	client := resty.New()
	client.SetBaseURL(serverURL)
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

	var bodyContent []byte
	bodyContent, err = json.Marshal(msg)
	if err != nil {
		log.Println(err)
		return
	}

	// если передан публичный ключ - шифровать сообщение
	if encoder != nil {
		bodyContent, err = encoder.Encode(bodyContent)
		if err != nil {
			log.Println(err)
		}
	}

	_, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(bodyContent).
		Post(`/update/`)
	if err != nil {
		log.Println(err)
		return
	}
}

// sendMetricsBatchByJSON отправляет словарь метрик Post запросом, в json формате.
func sendMetricsBatchByJSON(metrics map[string]interface{}) {
	var err error

	client := resty.New()
	client.SetBaseURL(serverURL)

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

	var bodyContent []byte
	bodyContent, err = json.Marshal(metricsToSend)
	if err != nil {
		log.Println(err)
		return
	}

	// если передан публичный ключ - шифровать сообщение
	if encoder != nil {
		bodyContent, err = encoder.Encode(bodyContent)
		if err != nil {
			log.Println(err)
		}
	}

	_, err = client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(bodyContent).
		Post(`/updates/`)
	if err != nil {
		log.Println(err)
		return
	}
}

func StopAgent() {
	// блокируем мьютекс обновления значений метрик
	updateMetricsMutex.Lock()
	// закрываем workpool
	WPool.Close()
}
