package agent

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestInitEnv(t *testing.T) {
	tests := []struct {
		name      string
		envVars   map[string]string
		wantEnv   Environment
		wantPanic bool
	}{
		{
			name:    "Test 1. No env vars.",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress: "localhost:8080", PollInterval: 2 * time.Second, ReportInterval: 10 * time.Second,
			},
			wantPanic: false,
		},
		{
			name: "Test 2. Correct env vars.",
			envVars: map[string]string{
				"ADDRESS": "localhost:3030", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "localhost:3030", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},

		{
			name: "Test 3. Incorrect env vars names.",
			envVars: map[string]string{
				"address": "localhost:8080", "REPORT_INTERVAL": "25s", "PollInterval": "2s",
			},
			wantEnv: Environment{
				ServerAddress: "localhost:8080", PollInterval: 2 * time.Second, ReportInterval: 25 * time.Second,
			},
			wantPanic: false,
		},
		{
			name: "Test 4. Incorrect ADDRESS env value.",
			envVars: map[string]string{
				"ADDRESS": "notUrl", "REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantEnv: Environment{
				ServerAddress: "notUrl", PollInterval: 5 * time.Second, ReportInterval: 20 * time.Second,
			},
			wantPanic: false,
		},
		{
			name: "Test 5. Incorrect env interval values.",
			envVars: map[string]string{
				"ADDRESS": "localhost:8080", "REPORT_INTERVAL": "20", "POLL_INTERVAL": "5s",
			},
			wantEnv:   Environment{},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// сброс влияния других тестов
			Env = Environment{}
			os.Clearenv()

			for key, value := range tt.envVars {
				err := os.Setenv(key, value)
				require.NoError(t, err)
			}

			if tt.wantPanic {
				assert.Panics(t, initEnv)
			} else {
				initEnv()
				assert.Equal(t, tt.wantEnv, Env)
			}
		})
	}
}

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

// todo: агента нужно тестировать только на отправление? Или вместе с ответом
func Test_sendMetric(t *testing.T) {
	int64Value, float64Value := int64(10), float64(12.133)
	metricCounter := message.Metrics{ID: "PollCount", MType: "counter", Value: nil, Delta: &int64Value}
	metricGauge := message.Metrics{ID: "RandomValue", MType: "gauge", Value: &float64Value, Delta: nil}

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
			Env.ServerAddress = svr.URL
			sendMetric(tt.args.paramName, tt.args.paramValue)
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
	Env.ServerAddress = svr.URL
	SendMetrics()
	assert.Lenf(t, gotMetricsReq, metricsCount, "Expected %d requests, got %d", metricsCount, len(gotMetricsReq))
}
