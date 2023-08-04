package server

import (
	"bytes"
	"compress/gzip"
	"context"
	"flag"
	"github.com/firesworder/devopsmetrics/internal/crypt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/filestore"
	"github.com/firesworder/devopsmetrics/internal/storage"
)

// Переменные для формирования состояния MemStorage
var metric1, metric2, metric3 *storage.Metric
var metric1upd20, metric2upd235, unknownMetric, unknownMetric2 *storage.Metric

func init() {
	metric1, _ = storage.NewMetric("PollCount", internal.CounterTypeName, int64(10))
	metric1upd20, _ = storage.NewMetric("PollCount", internal.CounterTypeName, int64(30))
	metric2, _ = storage.NewMetric("RandomValue", internal.GaugeTypeName, 12.133)
	metric2upd235, _ = storage.NewMetric("RandomValue", internal.GaugeTypeName, 23.5)
	metric3, _ = storage.NewMetric("Alloc", internal.GaugeTypeName, 7.77)
	unknownMetric, _ = storage.NewMetric("UnknownMetric", internal.CounterTypeName, int64(10))
	unknownMetric2, _ = storage.NewMetric("UnknownMetric", internal.GaugeTypeName, 7.77)
}

var testEnvVars = []string{
	"ADDRESS", "STORE_FILE", "STORE_INTERVAL", "RESTORE", "KEY", "DATABASE_DSN", "CRYPTO_KEY", "CONFIG", "TRUSTED_SUBNET",
}

func SaveOSVarsState(testEnvVars []string) map[string]string {
	osEnvVarsState := map[string]string{}
	for _, key := range testEnvVars {
		if v, ok := os.LookupEnv(key); ok {
			osEnvVarsState[key] = v
		}
	}
	return osEnvVarsState
}

func UpdateOSEnvState(t *testing.T, testEnvVars []string, newState map[string]string) {
	// удаляю переменные окружения, если они были до этого установлены
	for _, key := range testEnvVars {
		err := os.Unsetenv(key)
		require.NoError(t, err)
	}
	// устанавливаю переменные окружения использованные для теста
	for key, value := range newState {
		err := os.Setenv(key, value)
		require.NoError(t, err)
	}
}

// В рамках этой функции реализован и тест parseMetricParams, т.к. последнее является неотъемлимой
// частью ServeHTTP(выделана для лучшего восприятия)

type requestArgs struct {
	method      string
	url         string
	contentType string
	xRealIP     string
	body        string
}

type response struct {
	contentType string
	body        string
	statusCode  int
}

func compareMetricsState(t *testing.T, wantMS map[string]storage.Metric, mR storage.MetricRepository,
	ctx context.Context,
) {
	gotMS, err := mR.GetAll(ctx)
	require.NoError(t, err)
	assert.Equal(t, wantMS, gotMS)
}

func TestAddUpdateMetricHandler(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	tests := []struct {
		initState    map[string]storage.Metric
		wantedState  map[string]storage.Metric
		name         string
		request      requestArgs
		wantResponse response
	}{
		{
			name:         "Test 1. Correct request. Counter type. Add metric. Empty state",
			request:      requestArgs{url: `/update/counter/PollCount/10`, method: http.MethodPost},
			wantResponse: response{statusCode: http.StatusOK, contentType: "", body: ""},
			initState:    map[string]storage.Metric{},
			wantedState:  map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name:         "Test 2. Correct request. Counter type. Add metric. Filled state",
			request:      requestArgs{url: `/update/counter/PollCount/10`, method: http.MethodPost},
			wantResponse: response{statusCode: http.StatusOK, contentType: "", body: ""},
			initState: map[string]storage.Metric{
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name:         "Test 3. Correct request. Gauge type. Add metric. Empty state",
			request:      requestArgs{url: `/update/gauge/RandomValue/12.133`, method: http.MethodPost},
			wantResponse: response{statusCode: http.StatusOK, contentType: "", body: ""},
			initState:    map[string]storage.Metric{},
			wantedState:  map[string]storage.Metric{metric2.Name: *metric2},
		},
		{
			name:         "Test 4. Correct request. Gauge type. Add metric. Filled state",
			request:      requestArgs{url: `/update/gauge/RandomValue/12.133`, method: http.MethodPost},
			wantResponse: response{statusCode: http.StatusOK, contentType: "", body: ""},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name:         "Test 5. Correct request. Counter type. Update metric.",
			request:      requestArgs{url: `/update/counter/PollCount/20`, method: http.MethodPost},
			wantResponse: response{statusCode: http.StatusOK, contentType: "", body: ""},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1upd20.Name: *metric1upd20,
				metric3.Name:      *metric3,
			},
		},
		{
			name:         "Test 6. Correct request. Gauge type. Update metric.",
			request:      requestArgs{url: `/update/gauge/RandomValue/23.5`, method: http.MethodPost},
			wantResponse: response{statusCode: http.StatusOK, contentType: "", body: ""},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name:       *metric1,
				metric2upd235.Name: *metric2upd235,
			},
		},
		{
			name:    "Test 7. Incorrect http method.",
			request: requestArgs{url: `/update/counter/PollCount/10`, method: http.MethodPut},
			wantResponse: response{
				statusCode:  http.StatusMethodNotAllowed,
				contentType: "",
				body:        "",
			},
		},
		{
			name:    "Test 8. Incorrect url path(shorter).",
			request: requestArgs{url: `/update/counter/PollCount`, method: http.MethodPost},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
		},
		{
			name:    "Test 9. Incorrect url path(longer).",
			request: requestArgs{url: `/update/counter/PollCount/10/someinfo`, method: http.MethodPost},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
		},
		{
			name:    "Test 10. Incorrect metric type.",
			request: requestArgs{url: `/update/PollCount/RandomValue/10`, method: http.MethodPost},
			wantResponse: response{
				statusCode:  http.StatusNotImplemented,
				contentType: "text/plain; charset=utf-8",
				body:        "unhandled value type 'PollCount'\n",
			},
		},
		{
			name:    "Test 11. Incorrect metric value for metric type.",
			request: requestArgs{url: `/update/counter/PollCount/10.3`, method: http.MethodPost},
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "strconv.ParseInt: parsing \"10.3\": invalid syntax\n",
			},
		},
		{
			name:    "Test 12. Unknown metric.",
			request: requestArgs{url: `/update/counter/UnknownMetric/10`, method: http.MethodPost},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "",
				body:        "",
			},
			initState:   map[string]storage.Metric{},
			wantedState: map[string]storage.Metric{unknownMetric.Name: *unknownMetric},
		},
		{
			name:    "Test 13. Incorrect first part of URL.",
			request: requestArgs{url: `/updater/gauge/RandomValue/13.223`, method: http.MethodPost},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			statusCode, contentType, body := sendTestRequest(t, ts, tt.request)
			require.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponse.body, body)

			compareMetricsState(t, tt.wantedState, s.MetricStorage, context.Background())
		})
	}
}

func TestShowAllMetricsHandler(t *testing.T) {
	s := Server{}
	s.LayoutsDir = "./html_layouts/"
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	tests := []struct {
		memStorageState map[string]storage.Metric
		name            string
		request         requestArgs
		wantResponse    response
	}{
		{
			name:    "Test 1. Correct request, empty state.",
			request: requestArgs{method: http.MethodGet, url: "/"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "text/html; charset=utf-8",
			},
			memStorageState: map[string]storage.Metric{},
		},
		{
			name:    "Test 2. Correct request, with filled state.",
			request: requestArgs{method: http.MethodGet, url: "/"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "text/html; charset=utf-8",
			},
			memStorageState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name:    "Test 3. Incorrect method, empty state.",
			request: requestArgs{method: http.MethodPost, url: "/"},
			wantResponse: response{
				statusCode:  http.StatusMethodNotAllowed,
				contentType: "",
				body:        "",
			},
			memStorageState: map[string]storage.Metric{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.memStorageState)
			statusCode, contentType, body := sendTestRequest(t, ts, tt.request)
			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			if statusCode == http.StatusOK {
				assert.NotEmpty(t, body, "Empty body(html) response!")
			} else {
				assert.Equal(t, tt.wantResponse.body, body)
			}
		})
	}
}

func TestGetMetricHandler(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	filledState := map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
		metric3.Name: *metric3,
	}
	emptyState := map[string]storage.Metric{}

	tests := []struct {
		memStorageState map[string]storage.Metric
		name            string
		request         requestArgs
		wantResponse    response
	}{
		{
			name:    "Test 1. Correct url, empty state.",
			request: requestArgs{method: http.MethodGet, url: "/value/counter/PollCount"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "unknown metric\n",
			},
			memStorageState: emptyState,
		},
		{
			name:    "Test 2. Correct url, metric in filled state. Counter type",
			request: requestArgs{method: http.MethodGet, url: "/value/counter/PollCount"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "text/plain; charset=utf-8", body: "10",
			},
			memStorageState: filledState,
		},
		{
			name:    "Test 3. Correct url, metric in filled state. Gauge type",
			request: requestArgs{method: http.MethodGet, url: "/value/gauge/Alloc"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "text/plain; charset=utf-8", body: "7.77",
			},
			memStorageState: filledState,
		},
		{
			name:    "Test 4. Correct url, metric NOT in filled state.",
			request: requestArgs{method: http.MethodGet, url: "/value/gauge/AnotherMetric"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "unknown metric\n",
			},
			memStorageState: filledState,
		},
		{
			// Пока что я не проверяю типы, а только наличие метрики с соотв. названием
			// мб стоит дополнить. Хотя бы на проверку counter\gauge
			name:    "Test 5. Incorrect url. WrongType of metric",
			request: requestArgs{method: http.MethodGet, url: "/value/gauge/PollCount"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "text/plain; charset=utf-8", body: "10",
			},
			memStorageState: filledState,
		},
		{
			name:    "Test 6. Incorrect url. Skipped type part",
			request: requestArgs{method: http.MethodGet, url: "/value/PollCount"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			memStorageState: filledState,
		},
		{
			name:    "Test 7. Incorrect url. Skipped metricName part",
			request: requestArgs{method: http.MethodGet, url: "/value/counter/"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			memStorageState: filledState,
		},
		{
			name:    "Test 8. Incorrect url",
			request: requestArgs{method: http.MethodGet, url: "/val"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			memStorageState: filledState,
		},
		{
			name:            "Test 9. Correct url, but wrong method",
			request:         requestArgs{method: http.MethodPost, url: "/value/counter/PollCount"},
			wantResponse:    response{statusCode: http.StatusMethodNotAllowed, contentType: "", body: ""},
			memStorageState: filledState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.memStorageState)
			statusCode, contentType, body := sendTestRequest(t, ts, tt.request)
			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponse.body, body)
		})
	}
}

func TestAddUpdateMetricJSONHandler(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	tests := []struct {
		initState   map[string]storage.Metric
		wantedState map[string]storage.Metric
		name        string
		requestArgs
		wantResponse response
	}{
		{
			name: "Test correct counter #1. Add metric. Empty state",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
			initState:   map[string]storage.Metric{},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name: "Test correct counter #2. Add metric. Filled state",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
			initState: map[string]storage.Metric{
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name: "Test correct counter #3. Update metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":20}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":30}`,
			},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1upd20.Name: *metric1upd20,
				metric3.Name:      *metric3,
			},
		},
		{
			name: "Test correct counter #4. Unknown metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"UnknownMetric","type":"counter","delta":10}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"UnknownMetric","type":"counter","delta":10}`,
			},
			initState:   map[string]storage.Metric{},
			wantedState: map[string]storage.Metric{unknownMetric.Name: *unknownMetric},
		},

		{
			name: "Test correct gauge #1. Add metric. Empty state",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133}`,
			},
			initState:   map[string]storage.Metric{},
			wantedState: map[string]storage.Metric{metric2.Name: *metric2},
		},
		{
			name: "Test correct gauge #2. Add metric. Filled state",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133}`,
			},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name: "Test correct gauge #3. Update metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":23.5}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":23.5}`,
			},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name:       *metric1,
				metric2upd235.Name: *metric2upd235,
			},
		},
		{
			name: "Test correct gauge #4. Unknown metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"UnknownMetric","type":"gauge","value":7.77}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"UnknownMetric","type":"gauge","value":7.77}`,
			},
			initState:   map[string]storage.Metric{},
			wantedState: map[string]storage.Metric{unknownMetric2.Name: *unknownMetric2},
		},

		{
			name: "Test incorrect #1. Incorrect http method.",
			requestArgs: requestArgs{
				method:      http.MethodPut,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","value":10}`,
			},
			wantResponse: response{
				statusCode:  http.StatusMethodNotAllowed,
				contentType: "",
				body:        "",
			},
			initState:   map[string]storage.Metric{metric1.Name: *metric1},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name: "Test incorrect #2. Incorrect metric type.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"PollCount","value":10}`,
			},
			wantResponse: response{
				statusCode:  http.StatusNotImplemented,
				contentType: "text/plain; charset=utf-8",
				body:        "unhandled value type 'PollCount'\n",
			},
			initState:   map[string]storage.Metric{metric1.Name: *metric1},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name: "Test incorrect #3. Incorrect request body. Field value, but for type counter.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","value":10.3}`,
			},
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "param 'delta' cannot be nil for type 'counter'\n",
			},
			initState:   map[string]storage.Metric{metric1.Name: *metric1},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name: "Test incorrect #4. Incorrect request body. Field delta, but for type gauge.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"gauge","delta":10}`,
			},
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "param 'value' cannot be nil for type 'gauge'\n",
			},
			initState:   map[string]storage.Metric{metric1.Name: *metric1},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name: "Test incorrect #5. Incorrect metric value for metric type.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10.3}`,
			},
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "json: cannot unmarshal number 10.3 into Go struct field Metrics.delta of type int64\n",
			},
			initState:   map[string]storage.Metric{metric1.Name: *metric1},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name: "Test incorrect #6. Incorrect URL.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/updater",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","value":10.3}`,
			},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			initState:   map[string]storage.Metric{metric1.Name: *metric1},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			statusCode, contentType, body := sendTestRequest(t, ts, tt.requestArgs)

			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponse.body, body)

			compareMetricsState(t, tt.wantedState, s.MetricStorage, context.Background())
		})
	}
}

func TestAddUpdateMetricJSONHandlerWithHash(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	defaultEnv := Env

	tests := []struct {
		initState   map[string]storage.Metric
		wantedState map[string]storage.Metric
		name        string
		requestArgs
		wantResponse response
		env          environment
	}{
		{
			name: "Test correct counter #1. Add metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10,"hash":"4ca29a927a89931245cd4ad0782383d0fe0df883d31437cc5b85dc4dad3247c4"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10,"hash":"4ca29a927a89931245cd4ad0782383d0fe0df883d31437cc5b85dc4dad3247c4"}`,
			},
			initState: map[string]storage.Metric{
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name: "Test correct counter #2. Update metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":20,"hash":"a54ff39f2747a23c5834768f732d53719e143482400db980fcb886fc0a126faa"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":30,"hash":"84f056fb60dca6b2839556080bb2de533218121bebe8d95bf38e206479655d1a"}`,
			},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1upd20.Name: *metric1upd20,
				metric3.Name:      *metric3,
			},
		},

		{
			name: "Test correct gauge #1. Add metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133,"hash":"19742de723a08df1f3436d0b745ea7743c05520787cb32949497056fce1f7c70"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133,"hash":"19742de723a08df1f3436d0b745ea7743c05520787cb32949497056fce1f7c70"}`,
			},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name: "Test correct gauge #2. Update metric.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":23.5,"hash":"8dfae3f2574fadf10488b9104ad0d003d2267a8e045b22793c4e8c6b6f989d67"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":23.5,"hash":"8dfae3f2574fadf10488b9104ad0d003d2267a8e045b22793c4e8c6b6f989d67"}`,
			},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name:       *metric1,
				metric2upd235.Name: *metric2upd235,
			},
		},

		{
			name: "Test incorrect hash #1. Add counter metric, hash for different key.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10,"hash":"aaa29a927a89931245cd4ad0782383d0fe0df883d31437cc5b85dc4dad3247c4"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "hash is not correct\n",
			},
			initState: map[string]storage.Metric{
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
			wantedState: map[string]storage.Metric{
				metric2.Name: *metric2,
				metric3.Name: *metric3,
			},
		},
		{
			name: "Test incorrect hash #2. Update gauge metric, empty hash.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":23.5}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "hash is not correct\n",
			},
			initState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			},
			wantedState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Env = tt.env
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			statusCode, contentType, body := sendTestRequest(t, ts, tt.requestArgs)

			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponse.body, body)

			compareMetricsState(t, tt.wantedState, s.MetricStorage, context.Background())
		})
	}

	Env = defaultEnv
}

func sendTestRequest(t *testing.T, ts *httptest.Server, r requestArgs) (int, string, string) {
	// создаю реквест
	req, err := http.NewRequest(r.method, ts.URL+r.url, strings.NewReader(r.body))
	req.Header.Set("Content-Type", "application/json")
	if r.xRealIP != "" {
		req.Header.Set("X-Real-IP", r.xRealIP)
	}
	require.NoError(t, err)

	// делаю реквест на дефолтном клиенте
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// читаю ответ сервера
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp.StatusCode, resp.Header.Get("Content-Type"), string(respBody)
}

func TestGetMetricJSONHandler(t *testing.T) {
	filledState := map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
		metric3.Name: *metric3,
	}
	emptyState := map[string]storage.Metric{}

	// костыль, чтоб
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	tests := []struct {
		memStorageState map[string]storage.Metric
		name            string
		requestArgs
		wantResponse response
	}{
		{
			name: "Test correct counter #1. Correct request, metric is not present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter"}`,
			},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "metric with name 'PollCount' not found\n",
			},
			memStorageState: emptyState,
		},
		{
			name: "Test correct counter #2. Correct request, metric is present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter"}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
			memStorageState: filledState,
		},

		{
			name: "Test correct gauge #1. Correct request, metric is not present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
			},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "metric with name 'RandomValue' not found\n",
			},
			memStorageState: emptyState,
		},
		{
			name: "Test correct gauge #2. Correct request, metric is present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133}`,
			},
			memStorageState: filledState,
		},

		{
			name: "Test correct(?) others #1. Requested metric type differs with one in state",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"counter"}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133}`,
			},
			memStorageState: filledState,
		},
		{
			name: "Test correct(?) others #2. Unknown type",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"decimal"}`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
			memStorageState: filledState,
		},

		{
			name: "Test incorrect #1. Incorrect url.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/val",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
			},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			memStorageState: filledState,
		},
		{
			name: "Test incorrect #2. Wrong http method",
			requestArgs: requestArgs{
				method:      http.MethodGet,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
			},
			wantResponse:    response{statusCode: http.StatusMethodNotAllowed, contentType: "", body: ""},
			memStorageState: filledState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.memStorageState)
			statusCode, contentType, body := sendTestRequest(t, ts, tt.requestArgs)

			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponse.body, body)
		})
	}
}

func TestGetMetricJsonHandlerWithHash(t *testing.T) {
	filledState := map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
		metric3.Name: *metric3,
	}
	emptyState := map[string]storage.Metric{}

	// костыль, чтоб
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defaultEnv := Env
	defer ts.Close()

	tests := []struct {
		memStorageState map[string]storage.Metric
		name            string
		requestArgs
		wantResponse response
		env          environment
	}{
		// counter
		{
			name: "Test correct counter #1. Correct request, metric is not present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "metric with name 'PollCount' not found\n",
			},
			memStorageState: emptyState,
		},
		{
			name: "Test correct counter #2. Correct request, metric is present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10,"hash":"4ca29a927a89931245cd4ad0782383d0fe0df883d31437cc5b85dc4dad3247c4"}`,
			},
			memStorageState: filledState,
		},

		// gauge
		{
			name: "Test correct gauge #1. Correct request, metric is not present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "metric with name 'RandomValue' not found\n",
			},
			memStorageState: emptyState,
		},
		{
			name: "Test correct gauge #2. Correct request, metric is present.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
			},
			env: environment{Key: "Ayayaka"},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge","value":12.133,"hash":"19742de723a08df1f3436d0b745ea7743c05520787cb32949497056fce1f7c70"}`,
			},
			memStorageState: filledState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Env = tt.env
			s.MetricStorage = storage.NewMemStorage(tt.memStorageState)
			statusCode, contentType, body := sendTestRequest(t, ts, tt.requestArgs)

			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponse.body, body)
		})
	}
	// возвращаю Env до теста
	Env = defaultEnv
}

func TestServer_PingHandler(t *testing.T) {
	Env.DatabaseDsn = "postgresql://postgres:admin@localhost:5432/devops"
	s, err := NewServer()
	if err != nil {
		t.Skipf("cannot connect to db. db mocks are not ready yet")
	}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	reqArgs := requestArgs{
		method:      http.MethodGet,
		url:         "/ping",
		contentType: "text/plain",
		body:        ``,
	}

	tests := []struct {
		name string
		requestArgs
		wantResponse response
	}{
		{
			name: "Test 1. DB is accessible",
			wantResponse: response{
				statusCode: http.StatusOK,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			statusCode, _, _ := sendTestRequest(t, ts, reqArgs)

			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
		})
	}
}

func TestServer_InitFileStore(t *testing.T) {
	type ServerArgsPart struct {
		FileStore *filestore.FileStore
		StoreFile string
	}
	tests := []struct {
		wantFSArg       *filestore.FileStore
		name            string
		beforeInitSArgs ServerArgsPart
	}{
		{
			name: "Test #1. StoreFile field is not empty",
			beforeInitSArgs: ServerArgsPart{
				StoreFile: "some_file_path/file.json",
				FileStore: nil,
			},
			wantFSArg: &filestore.FileStore{StoreFilePath: "some_file_path/file.json"},
		},
		{
			name: "Test #2. StoreFile is set empty",
			beforeInitSArgs: ServerArgsPart{
				StoreFile: "",
				FileStore: nil,
			},
			wantFSArg: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				FileStore: tt.beforeInitSArgs.FileStore,
			}
			Env = environment{}
			Env.StoreFile = tt.beforeInitSArgs.StoreFile
			s.initFileStore()
			assert.Equal(t, tt.wantFSArg, s.FileStore)
		})
	}
}

func TestServer_InitMetricStorage(t *testing.T) {
	type serverArgs struct {
		FileStore *filestore.FileStore
		Restore   bool
	}

	tests := []struct {
		wantMetricStorage storage.MetricRepository
		name              string
		serverArgs
	}{
		{
			name: "Test #1. Restore=True and StoreFile exist and correct.",
			serverArgs: serverArgs{
				Restore:   true,
				FileStore: filestore.NewFileStore("file_storage_test/correct_ms_test.json"),
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			}),
		},
		{
			name: "Test #2. Restore=True and StoreFile path is empty.",
			serverArgs: serverArgs{
				Restore:   true,
				FileStore: nil,
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{}),
		},
		{
			name: "Test #3. Restore=True and StoreFile exist and incorrect(doesn't have MS in it).",
			serverArgs: serverArgs{
				Restore:   true,
				FileStore: filestore.NewFileStore("file_storage_test/not_ms_test.json"),
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{}),
		},
		{
			name: "Test #4. Restore=True and StoreFile not exist.",
			serverArgs: serverArgs{
				Restore:   true,
				FileStore: filestore.NewFileStore("not_existed.json"),
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{}),
		},

		{
			name: "Test #5. Restore=False and StoreFile exist and correct.",
			serverArgs: serverArgs{
				Restore:   false,
				FileStore: filestore.NewFileStore("file_storage_test/correct_ms_test.json"),
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{}),
		},
		{
			name: "Test #6. Restore=False and StoreFile path is empty.",
			serverArgs: serverArgs{
				Restore:   false,
				FileStore: nil,
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{}),
		},
		{
			name: "Test #7. Restore=False and StoreFile exist and and incorrect(doesn't have MS in it).",
			serverArgs: serverArgs{
				Restore:   false,
				FileStore: filestore.NewFileStore("file_storage_test/not_ms_test.json"),
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{}),
		},
		{
			name: "Test #8. Restore=False and StoreFile not exist.",
			serverArgs: serverArgs{
				Restore:   false,
				FileStore: filestore.NewFileStore("not_existed.json"),
			},
			wantMetricStorage: storage.NewMemStorage(map[string]storage.Metric{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverObj := Server{
				FileStore: tt.serverArgs.FileStore,
			}
			Env.Restore = tt.Restore
			serverObj.initMetricStorage()
			assert.Equal(t, tt.wantMetricStorage, serverObj.MetricStorage)
		})
	}
}

func TestParseEnvArgs(t *testing.T) {
	savedState := SaveOSVarsState(testEnvVars)
	envBefore := Env
	tests := []struct {
		name      string
		cmdStr    string
		envVars   map[string]string
		wantEnv   environment
		wantPanic bool
	}{
		{
			name:    "Test correct 1. Empty cmd args and env vars.",
			cmdStr:  "file.exe",
			envVars: map[string]string{},
			wantEnv: environment{
				ServerAddress: "localhost:8080",
				StoreInterval: 300 * time.Second,
				StoreFile:     "/tmp/devops-metrics-db.json",
				Restore:       true,
			},
			wantPanic: false,
		},
		{
			name:    "Test correct 2. Set cmd args and empty env vars.",
			cmdStr:  "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 20 * time.Second,
				StoreFile:     "somefile.json",
				Restore:       false,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 3. Empty cmd args and set env vars.",
			cmdStr: "file.exe",
			envVars: map[string]string{
				"ADDRESS": "env.site", "STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "env.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 4. Set cmd args and set env vars.",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{
				"ADDRESS": "env.site", "STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "env.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 5. Partially set cmd args and set env vars. Field ADDRESS",
			cmdStr: "file.exe -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{
				"ADDRESS": "env.site", "STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "env.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
			},
			wantPanic: false,
		},
		{
			name:   "Test correct 6. Set cmd args and partially set env vars. Field ADDRESS",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
			},
			wantPanic: false,
		},
		{
			name:   "Test 7. Field key, cmd",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false -k=ayayaka",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
				Key:           "ayayaka",
			},
			wantPanic: false,
		},
		{
			name:   "Test 8. Field key, env",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true", "KEY": "ayayaka",
			},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
				Key:           "ayayaka",
			},
			wantPanic: false,
		},
		{
			name:   "Test 9. Field key, not set",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
				Key:           "",
			},
			wantPanic: false,
		},

		{
			name:   "Test 10. Field DatabaseDsn, cmd",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false -d=localhost:5432",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
				DatabaseDsn:   "localhost:5432",
			},
			wantPanic: false,
		},
		{
			name:   "Test 11. Field key, env",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true", "DATABASE_DSN": "localhost:8080",
			},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
				DatabaseDsn:   "localhost:8080",
			},
			wantPanic: false,
		},
		{
			name:   "Test 12. Field key, not set",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
				DatabaseDsn:   "",
			},
			wantPanic: false,
		},
		// поле ConfigFilepath
		{
			name:   "Test 13. Field 'ConfigFilepath', set by cmd key 'c'. File exist.",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false -c=env_config_test.json",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "env.json",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "env_config_test.json",
				TrustedSubnet:      "192.168.1.1/24",
			},
			wantPanic: false,
		},
		{
			name:   "Test 14. Field 'ConfigFilepath', set by cmd key 'config'. File exist.",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false -config=env_config_test.json",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "env.json",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "env_config_test.json",
				TrustedSubnet:      "192.168.1.1/24",
			},
			wantPanic: false,
		},
		{
			name:   "Test 15. Field 'ConfigFilepath', set by env var 'CONFIG'. File exist.",
			cmdStr: "file.exe -a=cmd.site -i=20s -r=false",
			envVars: map[string]string{
				"STORE_INTERVAL": "60s", "CONFIG": "env_config_test.json", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "/path/to/file.db",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "env_config_test.json",
				TrustedSubnet:      "192.168.1.1/24",
			},
			wantPanic: false,
		},
		{
			name:   "Test 16. Field 'ConfigFilepath', set by env var 'CONFIG'. File not exist.",
			cmdStr: "file.exe --a=cmd.site -r=false --p=3s -config=env_config_test.json",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true", "CONFIG": "not_existed_config.json",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "env.json",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "not_existed_config.json",
			},
			wantPanic: true,
		},
		// поле TrustedSubnet
		{
			name:   "Test 17. Field 'TrustedSubnet', set by cmd key 't'.",
			cmdStr: "file.exe -a=cmd.site -i=20s -f=somefile.json -r=false -t=192.168.1.1/18",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "env.json",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "",
				TrustedSubnet:      "192.168.1.1/18",
			},
			wantPanic: false,
		},
		{
			name:   "Test 18. Field 'TrustedSubnet', set by env var 'TRUSTED_SUBNET'.",
			cmdStr: "file.exe -a=cmd.site -i=20s -r=false",
			envVars: map[string]string{
				"STORE_INTERVAL": "60s", "CONFIG": "env_config_test.json", "RESTORE": "true",
				"TRUSTED_SUBNET": "192.168.1.1/12",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "/path/to/file.db",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "env_config_test.json",
				TrustedSubnet:      "192.168.1.1/12",
			},
			wantPanic: false,
		},
		{
			name:   "Test 19. Field 'TrustedSubnet', set by config file field 'trusted_subnet'.",
			cmdStr: "file.exe -a=cmd.site -i=20s -r=false",
			envVars: map[string]string{
				"STORE_INTERVAL": "60s", "CONFIG": "env_config_test.json", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "/path/to/file.db",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "env_config_test.json",
				TrustedSubnet:      "192.168.1.1/24",
			},
			wantPanic: false,
		},
		{
			name:   "Test 20. Field 'TrustedSubnet', field not set.",
			cmdStr: "file.exe -a=cmd.site -i=20s -r=false",
			envVars: map[string]string{
				"STORE_FILE": "env.json", "STORE_INTERVAL": "60s", "RESTORE": "true",
			},
			wantEnv: environment{
				ServerAddress:      "cmd.site",
				StoreInterval:      60 * time.Second,
				StoreFile:          "env.json",
				Restore:            true,
				DatabaseDsn:        "",
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
			wantPanic: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаю env в дефолтные значения(обнулять я его не могу, т.к. flag линки потеряются)
			Env = environment{
				ServerAddress: "localhost:8080",
				StoreInterval: 300 * time.Second,
				StoreFile:     "/tmp/devops-metrics-db.json",
				Restore:       true,
				Key:           "",
				DatabaseDsn:   "",
				TrustedSubnet: "",
			}

			UpdateOSEnvState(t, testEnvVars, tt.envVars)
			// устанавливаю os.Args как эмулятор вызванной команды
			os.Args = strings.Split(tt.cmdStr, " ")
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.PanicOnError)
			initCmdArgs()

			// сама проверка корректности парсинга\получения ошибок
			if tt.wantPanic {
				assert.Panics(t, ParseEnvArgs)
			} else {
				ParseEnvArgs()
				assert.Equal(t, tt.wantEnv, Env)
			}
		})
	}
	Env = envBefore
	UpdateOSEnvState(t, testEnvVars, savedState)
}

// responseWC == response with compression(gzip), increment 8
type responseWC struct {
	response
	uncompressed bool
}

type requestArgsWC struct {
	requestArgs
	reqEncoding string
}

func sendTestRequestWithCompression(t *testing.T, ts *httptest.Server, r requestArgsWC) responseWC {
	var req *http.Request
	var err error
	if r.reqEncoding == "" {
		req, err = http.NewRequest(r.method, ts.URL+r.url, strings.NewReader(r.body))
	} else {
		// Compression
		var b bytes.Buffer
		w := gzip.NewWriter(&b)
		// запись данных
		_, err = w.Write([]byte(r.body))
		require.NoError(t, err)
		err = w.Close()
		require.NoError(t, err)
		req, err = http.NewRequest(r.method, ts.URL+r.url, &b)
		req.Header.Set("Content-Encoding", r.reqEncoding)
	}

	// создаю реквест
	req.Header.Add("Content-Type", "application/json")
	require.NoError(t, err)

	// делаю реквест на дефолтном клиенте
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	// читаю ответ сервера
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return responseWC{
		response: response{
			statusCode:  resp.StatusCode,
			contentType: resp.Header.Get("Content-Type"),
			body:        string(respBody),
		},
		uncompressed: resp.Uncompressed,
	}
}

func TestServer_gzipCompressor(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	tests := []struct {
		initState   map[string]storage.Metric
		wantedState map[string]storage.Metric
		name        string
		requestArgsWC
		wantResponse responseWC
	}{
		{
			name: "Test 1. Request for 'AddUpdateMetricJSONHandler'",
			requestArgsWC: requestArgsWC{
				requestArgs: requestArgs{
					method:      http.MethodPost,
					url:         "/update/",
					contentType: "application/json",
					body:        `{"id":"PollCount","type":"counter","delta":10}`,
				},
				reqEncoding: "",
			},
			wantResponse: responseWC{
				response: response{
					statusCode:  http.StatusOK,
					contentType: "application/json",
					body:        `{"id":"PollCount","type":"counter","delta":10}`,
				},
				uncompressed: true,
			},
			initState:   map[string]storage.Metric{},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			rWC := sendTestRequestWithCompression(t, ts, tt.requestArgsWC)
			require.Equal(t, tt.wantResponse.statusCode, rWC.statusCode)
			assert.Equal(t, tt.wantResponse.contentType, rWC.contentType)
			assert.Equal(t, tt.wantResponse.uncompressed, rWC.uncompressed)
			assert.Equal(t, tt.wantResponse.body, rWC.body)

			compareMetricsState(t, tt.wantedState, s.MetricStorage, context.Background())
		})
	}
}

func TestServer_gzipDecompressor(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	tests := []struct {
		initState   map[string]storage.Metric
		wantedState map[string]storage.Metric
		name        string
		requestArgsWC
		wantResponse responseWC
	}{
		{
			name: "Test 1. Request for 'AddUpdateMetricJSONHandler'",
			requestArgsWC: requestArgsWC{
				requestArgs: requestArgs{
					method:      http.MethodPost,
					url:         "/update/",
					contentType: "application/json",
					body:        `{"id":"PollCount","type":"counter","delta":10}`,
				},
				reqEncoding: "gzip",
			},
			wantResponse: responseWC{
				response: response{
					statusCode:  http.StatusOK,
					contentType: "application/json",
					body:        `{"id":"PollCount","type":"counter","delta":10}`,
				},
			},
			initState:   map[string]storage.Metric{},
			wantedState: map[string]storage.Metric{metric1.Name: *metric1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			rWC := sendTestRequestWithCompression(t, ts, tt.requestArgsWC)
			require.Equal(t, tt.wantResponse.statusCode, rWC.statusCode)
			assert.Equal(t, tt.wantResponse.contentType, rWC.contentType)
			assert.Equal(t, tt.wantResponse.body, rWC.body)

			compareMetricsState(t, tt.wantedState, s.MetricStorage, context.Background())
		})
	}
}

func TestServer_handlerBatchUpdate(t *testing.T) {
	ctx := context.Background()
	metricCounterFilled, _ := storage.NewMetric("CounterMetric", internal.CounterTypeName, int64(473771967))
	// 473771967(пред) + 247876521 = 721648488
	metricCounterUpdated, _ := storage.NewMetric("CounterMetric", internal.CounterTypeName, int64(721648488))
	// 721648488(пред) + 247876521 = 969525009
	metricCounterUpdatedMpl, _ := storage.NewMetric("CounterMetric", internal.CounterTypeName, int64(969525009))
	devTest := true
	Env.DatabaseDsn = "postgresql://postgres:admin@localhost:5432/devops"
	s, err := NewServer()
	if err != nil {
		devTest = false
		Env.DatabaseDsn = ""
		s, _ = NewServer()
	}
	ts := httptest.NewServer(s.newRouter())
	defer ts.Close()

	tests := []struct {
		memStorageState  map[string]storage.Metric
		wantStorageState map[string]storage.Metric
		name             string
		requestArgs
		wantResponse response
	}{
		{
			name: "Test 1. Empty metrics table.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/updates/",
				contentType: "application/json",
				body:        `[{"id":"PollCount","type":"counter","delta":10},{"id":"RandomValue","type":"gauge","value":12.133}]`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
			},
			memStorageState: map[string]storage.Metric{},
			wantStorageState: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			},
		},
		{
			name: "Test 2. FilledState.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/updates/",
				contentType: "application/json",
				body:        `[{"id":"CounterMetric","type":"counter","delta":247876521},{"id":"RandomValue","type":"gauge","value":23.5}]`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
			},
			memStorageState: map[string]storage.Metric{
				metric1.Name:             *metric1,
				metricCounterFilled.Name: *metricCounterFilled,
				metric2.Name:             *metric2,
			},
			wantStorageState: map[string]storage.Metric{
				metric1.Name:              *metric1,
				metricCounterUpdated.Name: *metricCounterUpdated,
				metric2upd235.Name:        *metric2upd235,
			},
		},
		{
			name: "Test 3. FilledState. Multiple metric in batch with the same name.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/updates/",
				contentType: "application/json",
				body:        `[{"id":"CounterMetric","type":"counter","delta":247876521},{"id":"RandomValue","type":"gauge","value":23.5},{"id":"CounterMetric","type":"counter","delta":247876521}]`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
			},
			memStorageState: map[string]storage.Metric{
				metric1.Name:             *metric1,
				metricCounterFilled.Name: *metricCounterFilled,
				metric2.Name:             *metric2,
			},
			wantStorageState: map[string]storage.Metric{
				metric1.Name:                 *metric1,
				metricCounterUpdatedMpl.Name: *metricCounterUpdatedMpl,
				metric2upd235.Name:           *metric2upd235,
			},
		},
		{
			name: "Test 4. Empty state. Multiple metric in batch with the same name.",
			requestArgs: requestArgs{
				method:      http.MethodPost,
				url:         "/updates/",
				contentType: "application/json",
				body:        `[{"id":"CounterMetric","type":"counter","delta":721648488},{"id":"RandomValue","type":"gauge","value":23.5},{"id":"CounterMetric","type":"counter","delta":247876521}]`,
			},
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
			},
			memStorageState: map[string]storage.Metric{},
			wantStorageState: map[string]storage.Metric{
				metricCounterUpdatedMpl.Name: *metricCounterUpdatedMpl,
				metric2upd235.Name:           *metric2upd235,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if devTest {
				_, err = s.DBConn.Exec("DELETE FROM metrics")
				require.NoError(t, err)
				for _, metric := range tt.memStorageState {
					err = s.MetricStorage.AddMetric(ctx, metric)
					require.NoError(t, err)
				}
			} else {
				s.MetricStorage = storage.NewMemStorage(tt.memStorageState)
			}

			statusCode, contentType, _ := sendTestRequest(t, ts, tt.requestArgs)
			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)

			compareMetricsState(t, tt.wantStorageState, s.MetricStorage, context.Background())
		})
	}
}

func TestServer_decryptMessage(t *testing.T) {
	testKeysDir := "../crypt/test/"

	s, err := NewServer()
	require.NoError(t, err)
	ts := httptest.NewServer(s.Router)
	defer ts.Close()

	testBody := `{"id":"RandomValue","type":"gauge","value":12.13}`
	// зашифрованное сообщение публичным ключом 1

	encoder, err := crypt.NewEncoder(testKeysDir + "publicKey_1_test.pem")
	require.NoError(t, err)
	encMsg, err := encoder.Encode([]byte(testBody))
	require.NoError(t, err)

	tests := []struct {
		name           string
		req            requestArgs
		privateKeyName string
		wantResponse   response
	}{
		{
			name: "Test 1. Encrypted msg, correct pair public+private key.",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
			},
			wantResponse: response{
				contentType: "application/json",
				statusCode:  http.StatusOK,
			},
			privateKeyName: "privateKey_1_test.pem",
		},
		{
			name: "Test 2. Encrypted msg, not correct pair public+private key.",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
			},
			wantResponse: response{
				contentType: "text/plain; charset=utf-8",
				statusCode:  http.StatusBadRequest,
			},
			privateKeyName: "privateKey_2_test.pem",
		},
		{
			name: "Test 3. Without encryption",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
			},
			wantResponse: response{
				contentType: "application/json",
				statusCode:  http.StatusOK,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// очищаю изменения в сторедже
			s.MetricStorage = &storage.MemStorage{Metrics: map[string]storage.Metric{}}
			// если указан приватный ключ - тестировать с шифрованием
			if tt.privateKeyName != "" {
				tt.req.body = string(encMsg)
				s.Decoder, err = crypt.NewDecoder(testKeysDir + tt.privateKeyName)
				require.NoError(t, err)
			} else {
				s.Decoder = nil
				tt.req.body = testBody
			}

			statusCode, contentType, body := sendTestRequest(t, ts, tt.req)

			require.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			if statusCode == http.StatusOK {
				// если хандлер отработал штатно, то тела ответа будет равно телу запроса
				assert.Equal(t, testBody, body)
			}
		})
	}
}

func TestServer_checkRequestSubnet(t *testing.T) {
	tests := []struct {
		name          string
		req           requestArgs
		trustedSubnet string
		wantResponse  response
	}{
		{
			name: "Test 1. TrustedSubnet field is not set. Update method.",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
				xRealIP:     "",
			},
			trustedSubnet: "",
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
		},
		{
			name: "Test 2. TrustedSubnet field is set. Get method. XRealIP is not set.",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
				xRealIP:     "",
			},
			trustedSubnet: "192.168.1.0/24",
			wantResponse: response{
				statusCode:  http.StatusForbidden,
				contentType: "text/plain; charset=utf-8",
				body:        "header value X-Real-IP can not be empty\n",
			},
		},
		{
			name: "Test 3. TrustedSubnet field is set. Update method. XRealIP is set, but IP invalid.",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
				xRealIP:     "192.122.1",
			},
			trustedSubnet: "192.168.1.0/24",
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "header value X-Real-IP is not valid\n",
			},
		},
		{
			name: "Test 4. TrustedSubnet field is set. Get method. XRealIP is set, but not in TrustedSubnet.",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/value/",
				contentType: "application/json",
				body:        `{"id":"RandomValue","type":"gauge"}`,
				xRealIP:     "192.122.1.1",
			},
			trustedSubnet: "192.168.1.0/24",
			wantResponse: response{
				statusCode:  http.StatusForbidden,
				contentType: "text/plain; charset=utf-8",
				body:        "user ip is not in trusted subnet\n",
			},
		},
		{
			name: "Test 5. TrustedSubnet field is set. Update method. XRealIP is set and in TrustedSubnet.",
			req: requestArgs{
				method:      http.MethodPost,
				url:         "/update/",
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
				xRealIP:     "192.168.1.1",
			},
			trustedSubnet: "192.168.1.0/24",
			wantResponse: response{
				statusCode:  http.StatusOK,
				contentType: "application/json",
				body:        `{"id":"PollCount","type":"counter","delta":10}`,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// подготовка к очередному тесту
			Env.TrustedSubnet = tt.trustedSubnet
			s, err := NewServer()
			require.NoError(t, err)
			s.MetricStorage = &storage.MemStorage{Metrics: map[string]storage.Metric{
				"RandomValue": {Value: float64(12.23), Name: "RandomValue"},
			}}
			ts := httptest.NewServer(s.Router)
			defer ts.Close()

			statusCode, contentType, respBody := sendTestRequest(t, ts, tt.req)
			require.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponse.body, respBody)
		})
	}
	// сбрасываю TrustedSubnet, чтобы не влиять на другие тесты
	Env.TrustedSubnet = ""
}

// Эти тесты должны быть внизу, т.к. вызывают гонку горутинами
// Тестирую изолированно только саму функцию(а не ее инъекции в обновл. MS хендлеры)
func TestServer_SyncSaveMetricStorage(t *testing.T) {
	type serverArgs struct {
		FileStore     *filestore.FileStore
		MetricStorage storage.MetricRepository
		StoreInterval time.Duration
	}
	tests := []struct {
		name       string
		wantFileAs string
		serverArgs
	}{
		{
			name: "Test #1. StoreInterval == 0 and FileStore != nil. MS != nil.",
			serverArgs: serverArgs{
				StoreInterval: 0,
				FileStore:     filestore.NewFileStore("file_storage_test/test_1.json"),
				MetricStorage: storage.NewMemStorage(map[string]storage.Metric{
					metric1.Name: *metric1,
					metric2.Name: *metric2,
				}),
			},
			wantFileAs: "file_storage_test/correct_ms_test.json",
		},
		{
			name: "Test #2. StoreInterval == 0 and FileStore == nil. MS != nil.",
			serverArgs: serverArgs{
				StoreInterval: 0,
				FileStore:     nil,
				MetricStorage: storage.NewMemStorage(map[string]storage.Metric{
					metric1.Name: *metric1,
					metric2.Name: *metric2,
				}),
			},
			wantFileAs: "",
		},
		{
			name: "Test #3. StoreInterval == 0 and FileStore != nil. MS == nil",
			serverArgs: serverArgs{
				StoreInterval: 0,
				FileStore:     filestore.NewFileStore("file_storage_test/test_3.json"),
				MetricStorage: nil,
			},
			wantFileAs: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				FileStore:     tt.serverArgs.FileStore,
				MetricStorage: tt.serverArgs.MetricStorage,
			}
			Env = environment{}
			Env.StoreInterval = tt.StoreInterval
			err := s.syncSaveMetricStorage()
			require.NoError(t, err)

			if tt.serverArgs.FileStore == nil {
				t.Skip("Не знаю как пока тестировать кейсы с FileStore == nil")
			}

			if tt.wantFileAs == "" {
				assert.NoFileExists(t, tt.serverArgs.FileStore.StoreFilePath)
			} else {
				sf := tt.serverArgs.FileStore.StoreFilePath
				filestore.AssertEqualFileContent(t, tt.wantFileAs, sf)

				// удаляю созданные сохранением файлы
				err = os.Remove(sf)
				require.NoError(t, err)
				// проверяем, что врем.файлы теста были удалены
				assert.NoFileExists(t, sf)
			}
		})
	}
}

func TestServer_InitRepeatableSave(t *testing.T) {
	type serverArgs struct {
		FileStore     *filestore.FileStore
		MetricStorage storage.MetricRepository
		StoreInterval time.Duration
	}
	tests := []struct {
		name string
		serverArgs
		wantFileAs string
	}{
		{
			name: "Test #1. StoreInterval > 0 and FileStore != nil. MS != nil.",
			serverArgs: serverArgs{
				StoreInterval: 500 * time.Millisecond,
				FileStore:     filestore.NewFileStore("file_storage_test/test_1.json"),
				MetricStorage: storage.NewMemStorage(map[string]storage.Metric{
					metric1.Name: *metric1,
					metric2.Name: *metric2,
				}),
			},
			wantFileAs: "file_storage_test/correct_ms_test.json",
		},
		{
			name: "Test #2. StoreInterval > 0 and FileStore == nil. MS != nil. Panic test",
			serverArgs: serverArgs{
				StoreInterval: 500 * time.Millisecond,
				FileStore:     nil,
				MetricStorage: storage.NewMemStorage(map[string]storage.Metric{
					metric1.Name: *metric1,
					metric2.Name: *metric2,
				}),
			},
			wantFileAs: "",
		},
		{
			name: "Test #3. StoreInterval > 0 and FileStore != nil. MS == nil",
			serverArgs: serverArgs{
				StoreInterval: 500 * time.Millisecond,
				FileStore:     filestore.NewFileStore("file_storage_test/test_3.json"),
				MetricStorage: nil,
			},
			wantFileAs: "",
		},
		// вариант StoreInterval == 0, FileStore != nil - это отдельная функция(и отдельно должна тестироваться)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &Server{
				FileStore:     tt.serverArgs.FileStore,
				MetricStorage: tt.serverArgs.MetricStorage,
			}
			Env.StoreInterval = tt.StoreInterval
			s.initRepeatableSave()
			time.Sleep(time.Second) // ждем пока тикер в initRepeatableSave отработает(горутиной)

			if tt.serverArgs.FileStore == nil {
				t.Skip("Не знаю как пока тестировать кейсы с FileStore == nil")
			}

			if tt.wantFileAs == "" {
				assert.NoFileExists(t, tt.serverArgs.FileStore.StoreFilePath)
			} else {
				sf := tt.serverArgs.FileStore.StoreFilePath
				filestore.AssertEqualFileContent(t, tt.wantFileAs, sf)

				// останавливаем горутину, чтобы она перестала писать файлы
				s.WriteTicker.Stop()
				time.Sleep(100 * time.Millisecond)
				// удаляю созданные сохранением файлы
				err := os.Remove(sf)
				require.NoError(t, err)
				// проверяем, что горутина остановилась и врем.файлы теста были удалены
				assert.NoFileExists(t, sf)
			}
		})
	}
}
