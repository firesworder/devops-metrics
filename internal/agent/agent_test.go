package agent

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestUpdateMetrics(t *testing.T) {
	runtime.ReadMemStats(&memstats)
	allocMetricBefore := memstats.Alloc
	pollCountBefore := PollCount
	randomValueBefore := RandomValue

	// нагрузка, чтобы повлиять на значения параметров в runtime.memstats
	demoSlice := []string{"demo"}
	for i := 0; i < 100; i++ {
		demoSlice = append(demoSlice, "demo")
	}

	UpdateMetrics()
	allocMetricAfter := memstats.Alloc
	pollCountAfter := PollCount
	randomValueAfter := RandomValue

	assert.NotEqual(t, allocMetricBefore, allocMetricAfter, "metric values were not updated")
	assert.Equal(t, true, pollCountBefore+1 == pollCountAfter,
		"PollCount was not updated correctly")
	assert.NotEqual(t, randomValueBefore, randomValueAfter, "RandomValue was not updated")
}

func TestSendMetricByURL(t *testing.T) {
	type args struct {
		paramName  string
		paramValue interface{}
	}
	tests := []struct {
		name           string
		args           args
		wantRequestURL string
	}{
		{
			name:           "Test 1. Gauge metric.",
			args:           args{paramName: "Alloc", paramValue: gauge(12.133)},
			wantRequestURL: "/update/gauge/Alloc/12.133000",
		},
		{
			name:           "Test 2. Counter metric.",
			args:           args{paramName: "PollCount", paramValue: counter(10)},
			wantRequestURL: "/update/counter/PollCount/10",
		},
		{
			name:           "Test 3. Metric with unknown type.",
			args:           args{paramName: "Alloc", paramValue: int64(10)},
			wantRequestURL: "",
		},
		{
			name:           "Test 4. Metric with nil value.",
			args:           args{paramName: "Alloc", paramValue: nil},
			wantRequestURL: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var actualRequestURL string
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				actualRequestURL = r.URL.Path
			}))
			defer svr.Close()
			ServerURL = svr.URL
			sendMetricByURL(tt.args.paramName, tt.args.paramValue)
			assert.Equal(t, tt.wantRequestURL, actualRequestURL)
		})
	}
}

func TestSendMetricByJson(t *testing.T) {
	int64Value, float64Value := int64(10), float64(12.133)
	metricCounter := message.Metrics{ID: "PollCount", MType: internal.CounterTypeName, Value: nil, Delta: &int64Value}
	metricGauge := message.Metrics{ID: "RandomValue", MType: internal.GaugeTypeName, Value: &float64Value, Delta: nil}

	type args struct {
		paramName  string
		paramValue interface{}
	}
	type wantRequest struct {
		contentType string
		msg         *message.Metrics
	}
	type wantResponse struct {
		statusCode  int
		contentType string
		msg         *message.Metrics
	}

	tests := []struct {
		name         string
		args         args
		wantRequest  *wantRequest
		wantResponse *wantResponse
	}{
		{
			name:        "Test 1. Gauge metric.",
			args:        args{paramName: "RandomValue", paramValue: gauge(12.133)},
			wantRequest: &wantRequest{contentType: "application/json", msg: &metricGauge},
		},
		{
			name:        "Test 2. Counter metric.",
			args:        args{paramName: "PollCount", paramValue: counter(10)},
			wantRequest: &wantRequest{contentType: "application/json", msg: &metricCounter},
		},
		{
			name:        "Test 3. Metric with unknown type.",
			args:        args{paramName: "Alloc", paramValue: int32(10)},
			wantRequest: nil,
		},
		{
			name:        "Test 4. Metric with nil value.",
			args:        args{paramName: "Alloc", paramValue: nil},
			wantRequest: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotRequest *wantRequest
			svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotRequest = &wantRequest{}
				gotRequest.contentType = r.Header.Get("Content-Type")

				msg := message.Metrics{}
				err := json.NewDecoder(r.Body).Decode(&msg)
				require.NoError(t, err, "cannot decode request body")
				gotRequest.msg = &msg
			}))
			defer svr.Close()
			ServerURL = svr.URL
			sendMetricByJson(tt.args.paramName, tt.args.paramValue)
			require.Equal(t, tt.wantRequest, gotRequest)
		})
	}
}

func TestSendMetrics(t *testing.T) {
	metricsCount := 29
	var gotMetricsReq = make([]string, 0, metricsCount)
	svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMetricsReq = append(gotMetricsReq, r.URL.Path)
	}))
	defer svr.Close()
	ServerURL = svr.URL
	SendMetrics()
	assert.Lenf(t, gotMetricsReq, metricsCount, "Expected %d requests, got %d", metricsCount, len(gotMetricsReq))
}

func TestParseEnvArgs(t *testing.T) {
	tests := []struct {
		name      string
		cmdStr    string
		envVars   map[string]string
		wantEnv   Environment
		wantPanic bool
	}{
		{
			name:    "Test correct 1. Empty cmd args and env vars.",
			cmdStr:  "file.exe",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress: "localhost:8080", PollInterval: 2 * time.Second, ReportInterval: 10 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:    "Test correct 2. Set cmd args and empty env vars.",
			cmdStr:  "file.exe --a=localhost:3030 -r=15s -p=3s",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress: "localhost:3030", PollInterval: 3 * time.Second, ReportInterval: 15 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 3. Empty cmd args and set env vars.",
			cmdStr: "file.exe",
			envVars: map[string]string{
				"ADDRESS": "localhost:3030", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "localhost:3030", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 4. Set cmd args and set env vars.",
			cmdStr: "file.exe --a=cmd.site -r=15s -p=3s",
			envVars: map[string]string{
				"ADDRESS": "env.site", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "env.site", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 5. Partially set cmd args and set env vars. Field ADDRESS",
			cmdStr: "file.exe --r=15s --p=3s",
			envVars: map[string]string{
				"ADDRESS": "env.site", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "env.site", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 6. Set cmd args and partially set env vars. Field ADDRESS",
			cmdStr: "file.exe --a=cmd.site --r=15s --p=3s",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "cmd.site", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаю env в дефолтные значения(обнулять я его не могу, т.к. flag линки потеряются)
			Env.ServerAddress = "localhost:8080"
			Env.ReportInterval = 10 * time.Second
			Env.PollInterval = 2 * time.Second

			// удаляю переменные окружения, если они были до этого установлены
			for _, key := range [3]string{"ADDRESS", "REPORT_INTERVAL", "POLL_INTERVAL"} {
				err := os.Unsetenv(key)
				require.NoError(t, err)
			}
			// устанавливаю переменные окружения использованные для теста
			for key, value := range tt.envVars {
				err := os.Setenv(key, value)
				require.NoError(t, err)
			}
			// устанавливаю os.Args как эмулятор вызванной команды
			os.Args = strings.Split(tt.cmdStr, " ")

			// сама проверка корректности парсинга\получения ошибок
			if tt.wantPanic {
				assert.Panics(t, ParseEnvArgs)
			} else {
				ParseEnvArgs()
				assert.Equal(t, tt.wantEnv, Env)
			}
		})
	}
}
