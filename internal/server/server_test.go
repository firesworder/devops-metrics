package server

import (
	"bytes"
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
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
	method string
	url    string
	body   *body
}

type body struct {
	content     []byte
	contentType string
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
			request: requestArgs{method: http.MethodGet, url: "/value/gauge/RandomValue"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "text/plain; charset=utf-8", body: "12.133",
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
			name:    "Test 8. Incorrect url. Only 'value' part",
			request: requestArgs{method: http.MethodGet, url: "/value"},
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
	// todo: смесь разных форматов записи
	floatVal := 12.33
	messageMetric1 := message.NewMetrics("PollCount", "counter", int64(10))
	messageMetric1upd := message.NewMetrics("PollCount", "counter", int64(20))
	messageMetric1upd20 := message.NewMetrics("PollCount", "counter", int64(30))
	messageMetric2 := message.NewMetrics("RandomValue", "gauge", 12.133)
	messageMetric2upd235 := message.NewMetrics("RandomValue", "gauge", 23.5)
	messageMetricUnknown := message.NewMetrics("UnknownMetric", "counter", int64(10))
	messageMetricUnknown2 := message.NewMetrics("UnknownMetric", "gauge", 7.77)
	messageIncorrectType := &message.Metrics{ID: "PollCount", MType: "unknown", Value: &floatVal, Delta: nil}
	messageIncorrectValueForType := &message.Metrics{ID: "PollCount", MType: "counter", Value: &floatVal, Delta: nil}

	s := NewServer()
	ts := httptest.NewServer(s.Router)
	defer ts.Close()

	const url = "/update"
	const method = http.MethodPost

	tests := []struct {
		name string
		requestArgs
		requestMessage      *message.Metrics
		wantResponse        response
		wantResponseMessage *message.Metrics
		initState           map[string]storage.Metric
		wantedState         map[string]storage.Metric
	}{
		{
			name:                "Test correct counter #1. Add metric. Empty state",
			requestMessage:      messageMetric1,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetric1,
			initState:           map[string]storage.Metric{},
			wantedState:         map[string]storage.Metric{metric1.Name: *metric1},
		},
		{
			name:                "Test correct counter #2. Add metric. Filled state",
			requestMessage:      messageMetric1,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetric1,
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
			name:                "Test correct counter #3. Update metric.",
			requestMessage:      messageMetric1upd,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetric1upd20,
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
			name:                "Test correct counter #4. Unknown metric.",
			requestMessage:      messageMetricUnknown,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetricUnknown,
			initState:           map[string]storage.Metric{},
			wantedState:         map[string]storage.Metric{unknownMetric.Name: *unknownMetric},
		},

		{
			name:                "Test correct gauge #1. Add metric. Empty state",
			requestMessage:      messageMetric2,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetric2,
			initState:           map[string]storage.Metric{},
			wantedState:         map[string]storage.Metric{metric2.Name: *metric2},
		},
		{
			name:                "Test correct gauge #2. Add metric. Filled state",
			requestMessage:      messageMetric2,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetric2,
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
			name:                "Test correct gauge #3. Update metric.",
			requestMessage:      messageMetric2upd235,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetric2upd235,
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
			name:                "Test correct gauge #4. Unknown metric.",
			requestMessage:      messageMetricUnknown2,
			wantResponse:        response{statusCode: http.StatusOK, contentType: "application/json"},
			wantResponseMessage: messageMetricUnknown2,
			initState:           map[string]storage.Metric{},
			wantedState:         map[string]storage.Metric{unknownMetric2.Name: *unknownMetric2},
		},

		// todo: реализовать отдельно тесты эти
		{
			name:           "Test incorrect #1. Incorrect http method.",
			requestArgs:    requestArgs{url: url, method: http.MethodPut},
			requestMessage: messageMetric1,
			wantResponse: response{
				statusCode:  http.StatusMethodNotAllowed,
				contentType: "",
				body:        "",
			},
			wantResponseMessage: nil,
		},
		{
			name:           "Test incorrect #2. Incorrect metric type.",
			requestMessage: messageIncorrectType,
			wantResponse: response{
				statusCode:  http.StatusNotImplemented,
				contentType: "text/plain; charset=utf-8",
				body:        "unhandled value type 'PollCount'\n",
			},
			wantResponseMessage: nil,
		},
		{
			name:           "Test incorrect #3. Incorrect metric value for metric type.",
			requestMessage: messageIncorrectValueForType,
			wantResponse: response{
				statusCode:  http.StatusBadRequest,
				contentType: "text/plain; charset=utf-8",
				body:        "strconv.ParseInt: parsing \"10.3\": invalid syntax\n",
			},
			wantResponseMessage: nil,
		},
		{
			name:           "Test incorrect #4. Incorrect URL.",
			requestArgs:    requestArgs{url: `/upd`, method: http.MethodPost},
			requestMessage: messageMetric1,
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			wantResponseMessage: nil,
		},
		// неправильно отправленный контент
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.initState)
			reqBody, err := json.Marshal(tt.requestMessage)
			require.NoError(t, err)
			tt.requestArgs.body = &body{contentType: "application/json", content: reqBody}

			if tt.requestArgs.method == "" {
				tt.requestArgs.method = method
			}
			if tt.requestArgs.url == "" {
				tt.requestArgs.url = url
			}
			statusCode, contentType, respBody := sendTestRequest(t, ts, tt.requestArgs)

			var respMessage *message.Metrics
			if statusCode == http.StatusOK {
				respMessage = &message.Metrics{}
				err = json.Unmarshal([]byte(respBody), &respMessage)
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponseMessage, respMessage)
			assert.Equal(t, tt.wantedState, s.MetricStorage.GetAll())
		})
	}
}

// todo: перевести на resty
func sendTestRequest(t *testing.T, ts *httptest.Server, r requestArgs) (int, string, string) {
	// создаю реквест
	var req *http.Request
	var err error
	if r.body != nil {
		bodyReader := bytes.NewReader(r.body.content)
		req, err = http.NewRequest(r.method, ts.URL+r.url, bodyReader)
		req.Header.Set("Content-Type", r.body.contentType)
	} else {
		req, err = http.NewRequest(r.method, ts.URL+r.url, nil)
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

// todo: в геттер(оба) добавить поиск метрики не только по имени, но и по типу
func TestGetMetricJSONHandler(t *testing.T) {
	counterMetricValue, gaugeMetricValue := int64(10), 12.133
	filledState := map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
		metric3.Name: *metric3,
	}
	emptyState := map[string]storage.Metric{}

	const url = "/value"
	const method = http.MethodPost

	s := NewServer()
	ts := httptest.NewServer(s.Router)
	defer ts.Close()

	tests := []struct {
		name                string
		request             requestArgs
		requestMessage      *message.Metrics
		wantResponse        response
		wantResponseMessage *message.Metrics
		memStorageState     map[string]storage.Metric
	}{
		{
			name:           "Test correct counter #1. Correct request, metric is not present.",
			request:        requestArgs{method: method, url: url},
			requestMessage: &message.Metrics{ID: "PollCount", MType: "counter"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "unknown metric\n",
			},
			wantResponseMessage: nil,
			memStorageState:     emptyState,
		},
		{
			name:           "Test correct counter #2. Correct request, metric is present.",
			request:        requestArgs{method: method, url: url},
			requestMessage: &message.Metrics{ID: "PollCount", MType: "counter"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "application/json",
			},
			wantResponseMessage: &message.Metrics{ID: "PollCount", MType: "counter", Delta: &counterMetricValue},
			memStorageState:     filledState,
		},

		{
			name:           "Test correct gauge #1. Correct request, metric is not present.",
			request:        requestArgs{method: method, url: url},
			requestMessage: &message.Metrics{ID: "RandomValue", MType: "gauge"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "unknown metric\n",
			},
			wantResponseMessage: nil,
			memStorageState:     emptyState,
		},
		{
			name:           "Test correct gauge #2. Correct request, metric is present.",
			request:        requestArgs{method: method, url: url},
			requestMessage: &message.Metrics{ID: "RandomValue", MType: "gauge"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "application/json",
			},
			wantResponseMessage: &message.Metrics{ID: "RandomValue", MType: "gauge", Value: &gaugeMetricValue},
			memStorageState:     filledState,
		},

		{
			// Пока что я не проверяю типы, а только наличие метрики с соотв. названием
			// мб стоит дополнить. Хотя бы на проверку counter\gauge
			name:           "Test correct(?) others #1. Requested metric type differs with one in state",
			request:        requestArgs{method: method, url: url},
			requestMessage: &message.Metrics{ID: "RandomValue", MType: "counter"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "application/json",
			},
			wantResponseMessage: &message.Metrics{ID: "RandomValue", MType: "gauge", Value: &gaugeMetricValue},
			memStorageState:     filledState,
		},
		{
			name:           "Test correct(?) others #1. Unknown type",
			request:        requestArgs{method: method, url: url},
			requestMessage: &message.Metrics{ID: "RandomValue", MType: "decimal"},
			wantResponse: response{
				statusCode: http.StatusOK, contentType: "application/json",
			},
			wantResponseMessage: &message.Metrics{ID: "RandomValue", MType: "gauge", Value: &gaugeMetricValue},
			memStorageState:     filledState,
		},

		{
			name:           "Test incorrect #1. Incorrect url.",
			request:        requestArgs{method: method, url: "/metric-value"},
			requestMessage: &message.Metrics{ID: "RandomValue", MType: "gauge"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			wantResponseMessage: nil,
			memStorageState:     filledState,
		},
		{
			name:           "Test incorrect #2. Metric name is not present",
			request:        requestArgs{method: method, url: url},
			requestMessage: &message.Metrics{ID: "SomeMetric", MType: "gauge"},
			wantResponse: response{
				statusCode:  http.StatusNotFound,
				contentType: "text/plain; charset=utf-8",
				body:        "404 page not found\n",
			},
			wantResponseMessage: nil,
			memStorageState:     filledState,
		},
		{
			name:                "Test incorrect #3. Wrong http method",
			request:             requestArgs{method: http.MethodPut, url: url},
			requestMessage:      &message.Metrics{ID: "RandomValue", MType: "gauge"},
			wantResponse:        response{statusCode: http.StatusMethodNotAllowed, contentType: "", body: ""},
			wantResponseMessage: nil,
			memStorageState:     filledState,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s.MetricStorage = storage.NewMemStorage(tt.memStorageState)
			reqBody, err := json.Marshal(tt.requestMessage)
			require.NoError(t, err)
			tt.request.body = &body{contentType: "application/json", content: reqBody}

			if tt.request.method == "" {
				tt.request.method = method
			}
			if tt.request.url == "" {
				tt.request.url = url
			}
			statusCode, contentType, respBody := sendTestRequest(t, ts, tt.request)

			var respMessage *message.Metrics
			if statusCode == http.StatusOK {
				respMessage = &message.Metrics{}
				err = json.Unmarshal([]byte(respBody), &respMessage)
				require.NoError(t, err)
			}

			require.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			assert.Equal(t, tt.wantResponseMessage, respMessage)

			assert.Equal(t, tt.wantResponse.statusCode, statusCode)
			assert.Equal(t, tt.wantResponse.contentType, contentType)
			//assert.Equal(t, tt.wantResponse.body, body)
		})
	}
}
