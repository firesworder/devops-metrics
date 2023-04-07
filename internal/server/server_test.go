package server

import (
	"bytes"
	"compress/gzip"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/filestore"
	"github.com/firesworder/devopsmetrics/internal/helper"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
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

// В рамках этой функции реализован и тест parseMetricParams, т.к. последнее является неотъемлимой
// частью ServeHTTP(выделана для лучшего восприятия)

type requestArgs struct {
	method      string
	url         string
	contentType string
	body        string
}

type response struct {
	statusCode  int
	contentType string
	body        string
}

func TestAddUpdateMetricHandler(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.NewRouter())
	defer ts.Close()

	tests := []struct {
		name         string
		request      requestArgs
		wantResponse response
		initState    map[string]storage.Metric
		wantedState  map[string]storage.Metric
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
			assert.Equal(t, tt.wantedState, s.MetricStorage.GetAll())
		})
	}
}

func TestShowAllMetricsHandler(t *testing.T) {
	s := Server{}
	s.LayoutsDir = "./html_layouts/"
	ts := httptest.NewServer(s.NewRouter())
	defer ts.Close()

	tests := []struct {
		name            string
		request         requestArgs
		wantResponse    response
		memStorageState map[string]storage.Metric
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
	ts := httptest.NewServer(s.NewRouter())
	defer ts.Close()

	filledState := map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
		metric3.Name: *metric3,
	}
	emptyState := map[string]storage.Metric{}

	tests := []struct {
		name            string
		request         requestArgs
		wantResponse    response
		memStorageState map[string]storage.Metric
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
				statusCode: http.StatusOK, contentType: "text/plain; charset=utf-8", body: "7.770",
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
	ts := httptest.NewServer(s.NewRouter())
	defer ts.Close()

	tests := []struct {
		name string
		requestArgs
		wantResponse response
		initState    map[string]storage.Metric
		wantedState  map[string]storage.Metric
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

		// todo: реализовать отдельно тесты эти
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
			assert.Equal(t, tt.wantedState, s.MetricStorage.GetAll())
		})
	}
}

// todo: перевести на resty
func sendTestRequest(t *testing.T, ts *httptest.Server, r requestArgs) (int, string, string) {
	// создаю реквест
	req, err := http.NewRequest(r.method, ts.URL+r.url, strings.NewReader(r.body))
	req.Header.Set("Content-Type", "application/json")
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

// todo: в геттер(оба) добавить поиск метрики не только по имени, но и по типу
func TestGetMetricJSONHandler(t *testing.T) {
	filledState := map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
		metric3.Name: *metric3,
	}
	emptyState := map[string]storage.Metric{}

	// костыль, чтоб
	s := Server{}
	ts := httptest.NewServer(s.NewRouter())
	defer ts.Close()

	tests := []struct {
		name string
		requestArgs
		wantResponse    response
		memStorageState map[string]storage.Metric
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

		// todo: добавить проверку типов? Не просто так ведь передается(придется сильно рефакторить)
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

func TestServer_InitFileStore(t *testing.T) {
	type ServerArgsPart struct {
		StoreFile string
		FileStore *filestore.FileStore
	}
	tests := []struct {
		name            string
		beforeInitSArgs ServerArgsPart
		wantFSArg       *filestore.FileStore
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
			// todo: проверить, что это устанавливается значение(пустое), хз как envDefault среагирует
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
			Env = Environment{}
			Env.StoreFile = tt.beforeInitSArgs.StoreFile
			s.InitFileStore()
			assert.Equal(t, tt.wantFSArg, s.FileStore)
		})
	}
}

func TestServer_InitMetricStorage(t *testing.T) {
	type serverArgs struct {
		Restore   bool
		FileStore *filestore.FileStore
	}

	tests := []struct {
		name string
		serverArgs
		wantMetricStorage storage.MetricRepository
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
			serverObj.InitMetricStorage()
			assert.Equal(t, tt.wantMetricStorage, serverObj.MetricStorage)
		})
	}
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
			wantEnv: Environment{
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
			wantEnv: Environment{
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
			wantEnv: Environment{
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
			wantEnv: Environment{
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
			wantEnv: Environment{
				ServerAddress: "cmd.site",
				StoreInterval: 60 * time.Second,
				StoreFile:     "env.json",
				Restore:       true,
			},
			wantPanic: false,
		},
		// todo: описать более детально тесткейсы(пока только самые простые)
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаю env в дефолтные значения(обнулять я его не могу, т.к. flag линки потеряются)
			Env = Environment{
				ServerAddress: "localhost:8080",
				StoreInterval: 300 * time.Second,
				StoreFile:     "/tmp/devops-metrics-db.json",
				Restore:       true,
			}

			// удаляю переменные окружения, если они были до этого установлены
			for _, key := range [4]string{"ADDRESS", "STORE_FILE", "STORE_INTERVAL", "RESTORE"} {
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
	ts := httptest.NewServer(s.NewRouter())
	defer ts.Close()

	tests := []struct {
		name string
		requestArgsWC
		wantResponse responseWC
		initState    map[string]storage.Metric
		wantedState  map[string]storage.Metric
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
		// todo: добавить кейс, когда агент не хочет сжатия
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			rWC := sendTestRequestWithCompression(t, ts, tt.requestArgsWC)
			require.Equal(t, tt.wantResponse.statusCode, rWC.statusCode)
			assert.Equal(t, tt.wantResponse.contentType, rWC.contentType)
			assert.Equal(t, tt.wantResponse.uncompressed, rWC.uncompressed)
			assert.Equal(t, tt.wantResponse.body, rWC.body)
			assert.Equal(t, tt.wantedState, s.MetricStorage.GetAll())
		})
	}
}

func TestServer_gzipDecompressor(t *testing.T) {
	s := Server{}
	ts := httptest.NewServer(s.NewRouter())
	defer ts.Close()

	tests := []struct {
		name string
		requestArgsWC
		wantResponse responseWC
		initState    map[string]storage.Metric
		wantedState  map[string]storage.Metric
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
		// todo: добавить тестов
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			rWC := sendTestRequestWithCompression(t, ts, tt.requestArgsWC)
			require.Equal(t, tt.wantResponse.statusCode, rWC.statusCode)
			assert.Equal(t, tt.wantResponse.contentType, rWC.contentType)
			assert.Equal(t, tt.wantResponse.body, rWC.body)
			assert.Equal(t, tt.wantedState, s.MetricStorage.GetAll())
		})
	}
}

// todo: тесты ниже исправить. До тех пор - оставлять их в самом конце(устраивают гонку горутинами)
// Тестирую изолированно только саму функцию(а не ее инъекции в обновл. MS хендлеры)
func TestServer_SyncSaveMetricStorage(t *testing.T) {
	type serverArgs struct {
		StoreInterval time.Duration
		FileStore     *filestore.FileStore
		MetricStorage storage.MetricRepository
	}
	tests := []struct {
		name string
		serverArgs
		wantFileAs string
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
		// todo: тест на появление\отсутствие паники, по сути.
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
			Env = Environment{}
			Env.StoreInterval = tt.StoreInterval
			err := s.SyncSaveMetricStorage()
			require.NoError(t, err)

			if tt.serverArgs.FileStore == nil {
				t.Skip("Не знаю как пока тестировать кейсы с FileStore == nil")
			}

			if tt.wantFileAs == "" {
				assert.NoFileExists(t, tt.serverArgs.FileStore.StoreFilePath)
			} else {
				sf := tt.serverArgs.FileStore.StoreFilePath
				helper.AssertEqualFileContent(t, tt.wantFileAs, sf)

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
		StoreInterval time.Duration
		FileStore     *filestore.FileStore
		MetricStorage storage.MetricRepository
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
		// todo: тест на появление\отсутствие паники, по сути.
		{
			name: "Test #2. StoreInterval > 0 and FileStore == nil. MS != nil.",
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
			s.InitRepeatableSave()
			time.Sleep(time.Second) // ждем пока тикер в InitRepeatableSave отработает(горутиной)

			if tt.serverArgs.FileStore == nil {
				t.Skip("Не знаю как пока тестировать кейсы с FileStore == nil")
			}

			if tt.wantFileAs == "" {
				assert.NoFileExists(t, tt.serverArgs.FileStore.StoreFilePath)
			} else {
				sf := tt.serverArgs.FileStore.StoreFilePath
				helper.AssertEqualFileContent(t, tt.wantFileAs, sf)

				// останавливаем горутину, чтобы она перестала писать файлы
				s.WriteTicker.Stop()
				time.Sleep(100 * time.Millisecond)
				// удаляю созданные сохранением файлы
				err = os.Remove(sf)
				require.NoError(t, err)
				// проверяем, что горутина остановилась и врем.файлы теста были удалены
				assert.NoFileExists(t, sf)
			}
		})
	}
}
