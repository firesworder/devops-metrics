package server

import (
	"errors"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"net/http"
	"strconv"
	"strings"
)

type errorHTTP struct {
	message    string
	statusCode int
}

type MetricReqHandler struct {
	rootURLPath   string
	method        string
	urlPathLen    int
	MetricStorage storage.MetricRepository
}

func NewDefaultMetricHandler() MetricReqHandler {
	return MetricReqHandler{
		rootURLPath: "update", method: http.MethodPost, urlPathLen: 4, MetricStorage: storage.MetricStorage,
	}
}

func (mrh MetricReqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metric, err := mrh.parseMetricParams(r)
	if err != nil {
		http.Error(w, err.message, err.statusCode)
		return
	}
	errorObj := mrh.MetricStorage.UpdateOrAddMetric(*metric)
	if errorObj != nil {
		http.Error(w, errorObj.Error(), http.StatusBadRequest)
		return
	}
}

func (mrh MetricReqHandler) parseMetricParams(r *http.Request) (m *storage.Metric, err *errorHTTP) {
	if r.Method != mrh.method {
		err = &errorHTTP{message: "Only POST method allowed", statusCode: http.StatusMethodNotAllowed}
		return
	}

	urlParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(urlParts) != mrh.urlPathLen {
		err = &errorHTTP{
			message: fmt.Sprintf(
				"Некорректный URL запроса. Ожидаемое число частей пути URL: 4, получено %d", len(urlParts)),
			statusCode: http.StatusBadRequest,
		}
		return
	}
	rootURLPath, typeName, paramName, paramValueStr := urlParts[0], urlParts[1], urlParts[2], urlParts[3]
	if rootURLPath != mrh.rootURLPath {
		err = &errorHTTP{
			message: fmt.Sprintf(
				"Некорректный URL запроса. Ожидаемая первая часть пути '%s', получено '%s'",
				mrh.rootURLPath, rootURLPath),
			statusCode: http.StatusBadRequest,
		}
		return
	}

	var paramValue interface{}
	var parseErr error
	switch typeName {
	case "counter":
		paramValue, parseErr = strconv.ParseInt(paramValueStr, 10, 64)
	case "gauge":
		paramValue, parseErr = strconv.ParseFloat(paramValueStr, 64)
	default:
		parseErr = errors.New("неизвестный тип метрики")
	}
	if parseErr != nil {
		err = &errorHTTP{
			message: fmt.Sprintf(
				"Ошибка приведения значения '%s' метрики к типу '%s'", paramValueStr, typeName),
			statusCode: http.StatusBadRequest,
		}
		return
	}

	m, metricError := storage.NewMetric(paramName, typeName, paramValue)
	if metricError != nil {
		err = &errorHTTP{
			message:    metricError.Error(),
			statusCode: http.StatusBadRequest,
		}
		return
	}

	return
}
