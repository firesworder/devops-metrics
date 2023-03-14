package agent

import (
	"fmt"
	"github.com/stretchr/testify/assert"
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

func Test_sendMetric(t *testing.T) {
	type args struct {
		paramName  string
		paramValue interface{}
	}
	tests := []struct {
		name           string
		args           args
		wantRequestURL string
		wantError      error
	}{
		{
			name:           "Test 1. Gauge metric.",
			args:           args{paramName: "Alloc", paramValue: gauge(12.133)},
			wantRequestURL: "/update/gauge/Alloc/12.133000",
			wantError:      nil,
		},
		{
			name:           "Test 2. Counter metric.",
			args:           args{paramName: "PollCount", paramValue: counter(10)},
			wantRequestURL: "/update/counter/PollCount/10",
			wantError:      nil,
		},
		{
			name:           "Test 3. Metric with unknown type.",
			args:           args{paramName: "Alloc", paramValue: int64(10)},
			wantRequestURL: "",
			wantError:      fmt.Errorf("unhandled metric type '%T'", int64(10)),
		},
		{
			name:           "Test 4. Metric with nil value.",
			args:           args{paramName: "Alloc", paramValue: nil},
			wantRequestURL: "",
			wantError:      fmt.Errorf("unhandled metric type '%T'", nil),
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
			err := sendMetric(tt.args.paramName, tt.args.paramValue)
			assert.Equal(t, tt.wantRequestURL, actualRequestURL)
			assert.Equal(t, tt.wantError, err)
		})
	}
}
