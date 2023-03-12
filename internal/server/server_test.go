package server

import (
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// В рамках этой функции реализован и тест parseMetricParams, т.к. последнее является неотъемлимой
// частью ServeHTTP(выделана для лучшего восприятия)
func TestMetricReqHandler_ServeHTTP(t *testing.T) {
	type request struct {
		url    string
		method string
	}
	type response struct {
		statusCode int
		body       string
	}
	type metricArgs struct {
		name     string
		typeName string
		rawValue interface{}
	}
	tests := []struct {
		name         string
		request      request
		wantResponse response
		wantMetric   metricArgs
	}{
		{
			name:         "Test 1. Correct request(counter).",
			request:      request{url: `/update/counter/PollCount/10`, method: http.MethodPost},
			wantResponse: response{statusCode: http.StatusOK, body: ""},
			wantMetric:   metricArgs{name: "PollCount", typeName: "counter", rawValue: int64(10)},
		},
		{
			name:    "Test 2. Incorrect http method.",
			request: request{url: `/update/counter/PollCount/10`, method: http.MethodGet},
			wantResponse: response{
				statusCode: http.StatusMethodNotAllowed,
				body:       "Only POST method allowed",
			},
			wantMetric: metricArgs{},
		},
		{
			name:    "Test 3. Incorrect url path(shorter).",
			request: request{url: `/update/counter/PollCount`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusBadRequest,
				body:       "Некорректный URL запроса. Ожидаемое число частей пути URL: 4, получено 3",
			},
			wantMetric: metricArgs{},
		},
		{
			name:    "Test 4. Incorrect url path(longer).",
			request: request{url: `/update/counter/PollCount/10/someinfo`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusBadRequest,
				body:       "Некорректный URL запроса. Ожидаемое число частей пути URL: 4, получено 5",
			},
			wantMetric: metricArgs{},
		},
		{
			// todo: у меня обработки такой ошибки нет, надо добавить!
			name:    "Test 5. Incorrect url order.",
			request: request{url: `/update/PollCount/counter/10`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusBadRequest,
				body:       "Ошибка приведения значения '10' метрики к типу 'PollCount'",
			},
			wantMetric: metricArgs{},
		},
		{
			name:    "Test 6. Unknown metric type.",
			request: request{url: `/update/integer/PollCount/10`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusBadRequest,
				body:       "Ошибка приведения значения '10' метрики к типу 'integer'",
			},
			wantMetric: metricArgs{},
		},
		{
			name:    "Test 7. Unknown metric type.",
			request: request{url: `/update/integer/PollCount/10`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusBadRequest,
				body:       "Ошибка приведения значения '10' метрики к типу 'integer'",
			},
			wantMetric: metricArgs{},
		},
		{
			name:    "Test 8. Incorrect metric value for metric type.",
			request: request{url: `/update/counter/PollCount/10.3`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusBadRequest,
				body:       "Ошибка приведения значения '10.3' метрики к типу 'counter'",
			},
			wantMetric: metricArgs{},
		},
		{
			name:    "Test 9. Unknown metric.",
			request: request{url: `/update/counter/SomeMetric/10`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusOK,
				body:       "",
			},
			wantMetric: metricArgs{name: "SomeMetric", typeName: "counter", rawValue: int64(10)},
		},
		{
			name:    "Test 10. Correct gauge type metric.",
			request: request{url: `/update/gauge/RandomValue/13.223`, method: http.MethodPost},
			wantResponse: response{
				statusCode: http.StatusOK,
				body:       "",
			},
			wantMetric: metricArgs{name: "RandomValue", typeName: "gauge", rawValue: 13.223},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// тестовый запрос
			tr := httptest.NewRequest(tt.request.method, tt.request.url, nil)
			// тестовый писатель
			trw := httptest.NewRecorder()
			// handler
			h := NewDefaultMetricHandler()
			h.MetricStorage = storage.NewMemStorage(map[string]storage.Metric{})
			h.ServeHTTP(trw, tr)
			// получаю респонс из писателя
			tResponse := trw.Result()

			defer tResponse.Body.Close()
			tBody, err := io.ReadAll(tResponse.Body)
			if assert.NoError(t, err) {
				assert.Equal(t, tt.wantResponse.body, strings.TrimSpace(string(tBody)))
			}
			// если статус ответа запроса отличается - смысла проверять добавление метрики в стейт нет
			require.Equal(t, tt.wantResponse.statusCode, tResponse.StatusCode)

			if tResponse.StatusCode == http.StatusOK {
				metric, err := storage.NewMetric(tt.wantMetric.name, tt.wantMetric.typeName, tt.wantMetric.rawValue)
				wantStorage := storage.NewMemStorage(map[string]storage.Metric{tt.wantMetric.name: *metric})
				assert.NoError(t, err)
				assert.Equal(t, wantStorage, h.MetricStorage)
			}
		})
	}
}
