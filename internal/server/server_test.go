package server

import (
	"github.com/firesworder/devopsmetrics/internal/file_store"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Переменные для формирования состояния MemStorage
var metric1, metric2, metric3 *storage.Metric
var metric1upd20, metric2upd235, unknownMetric, unknownMetric2 *storage.Metric

func init() {
	metric1, _ = storage.NewMetric("PollCount", "counter", int64(10))
	metric1upd20, _ = storage.NewMetric("PollCount", "counter", int64(30))
	metric2, _ = storage.NewMetric("RandomValue", "gauge", 12.133)
	metric2upd235, _ = storage.NewMetric("RandomValue", "gauge", 23.5)
	metric3, _ = storage.NewMetric("Alloc", "gauge", 7.77)
	unknownMetric, _ = storage.NewMetric("UnknownMetric", "counter", int64(10))
	unknownMetric2, _ = storage.NewMetric("UnknownMetric", "gauge", 7.77)
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
	s := NewServer()
	ts := httptest.NewServer(s.Router)
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
	s := NewServer()
	s.LayoutsDir = "./html_layouts/"
	ts := httptest.NewServer(s.Router)
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
	s := NewServer()
	ts := httptest.NewServer(s.Router)
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
	s := NewServer()
	ts := httptest.NewServer(s.Router)
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

	s := NewServer()
	ts := httptest.NewServer(s.Router)
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

// todo: обновить, т.к. функция изменилась
func TestServer_initServerAddress(t *testing.T) {
	tests := []struct {
		name              string
		envs              map[string]string
		wantServerAddress string
	}{
		{
			name:              "Test 1. Env serverAddress is not set.",
			envs:              map[string]string{},
			wantServerAddress: "localhost:8080",
		},
		{
			name:              "Test 2. Env serverAddress is set.",
			envs:              map[string]string{"ADDRESS": "localhost:3030"},
			wantServerAddress: "localhost:8080",
		},
		{
			name:              "Test 3. Empty env serverAddress.",
			envs:              map[string]string{"ADDRESS": ""},
			wantServerAddress: "localhost:8080",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := Server{}
			srv.initEnvParams()
			assert.Equal(t, tt.wantServerAddress, srv.ServerAddress)
		})
	}
}

func TestServer_InitFileStore(t *testing.T) {
	type ServerArgsPart struct {
		StoreFile string
		FileStore *file_store.FileStore
	}
	tests := []struct {
		name            string
		beforeInitSArgs ServerArgsPart
		wantFSArg       *file_store.FileStore
	}{
		{
			name: "Test #1. StoreFile field is not empty",
			beforeInitSArgs: ServerArgsPart{
				StoreFile: "some_file_path/file.json",
				FileStore: nil,
			},
			wantFSArg: &file_store.FileStore{StoreFilePath: "some_file_path/file.json"},
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
				StoreFile: tt.beforeInitSArgs.StoreFile,
				FileStore: tt.beforeInitSArgs.FileStore,
			}
			s.InitFileStore()
			assert.Equal(t, tt.wantFSArg, s.FileStore)
		})
	}
}
