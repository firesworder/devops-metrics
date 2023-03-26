package agent

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
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
			ServerURL = svr.URL
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
	ServerURL = svr.URL
	SendMetrics()
	assert.Lenf(t, gotMetricsReq, metricsCount, "Expected %d requests, got %d", metricsCount, len(gotMetricsReq))
}
