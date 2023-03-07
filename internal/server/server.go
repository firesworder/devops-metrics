package server

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

var metricsTypes = map[string]string{
	"Alloc":         "gauge",
	"BuckHashSys":   "gauge",
	"Frees":         "gauge",
	"GCCPUFraction": "gauge",
	"GCSys":         "gauge",
	"HeapAlloc":     "gauge",
	"HeapIdle":      "gauge",
	"HeapInuse":     "gauge",
	"HeapObjects":   "gauge",
	"HeapReleased":  "gauge",
	"HeapSys":       "gauge",
	"LastGC":        "gauge",
	"Lookups":       "gauge",
	"MCacheInuse":   "gauge",
	"MCacheSys":     "gauge",
	"MSpanInuse":    "gauge",
	"MSpanSys":      "gauge",
	"Mallocs":       "gauge",
	"NextGC":        "gauge",
	"NumForcedGC":   "gauge",
	"NumGC":         "gauge",
	"OtherSys":      "gauge",
	"PauseTotalNs":  "gauge",
	"StackInuse":    "gauge",
	"StackSys":      "gauge",
	"Sys":           "gauge",
	"TotalAlloc":    "gauge",
	"PollCount":     "counter",
	"RandomValue":   "gauge",
}

type errorHTTP struct {
	message    string
	statusCode int
}

type MetricReqHandler struct {
	rootURLPath string
	method      string
	urlPathLen  int
}

type metric struct {
	typeName, paramName string
	paramValue          interface{}
}

func NewDefaultMetricHandler() MetricReqHandler {
	return MetricReqHandler{rootURLPath: "update", method: http.MethodPost, urlPathLen: 4}
}

func (mrh MetricReqHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	metricObj, err := mrh.parseMetricParams(r)
	if err != nil {
		http.Error(w, err.message, err.statusCode)
		return
	}

	if metricObj.typeName == "counter" {
		fmt.Printf("Type: %s | Param: %s | Value: %d\n", metricObj.typeName, metricObj.paramName, metricObj.paramValue)
	} else if metricObj.typeName == "gauge" {
		fmt.Printf("Type: %s | Param: %s | Value: %f\n", metricObj.typeName, metricObj.paramName, metricObj.paramValue)
	}
}

func (mrh MetricReqHandler) parseMetricParams(r *http.Request) (m *metric, err *errorHTTP) {
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
				"Некорректный URL запроса. Ожидаемая первая часть пути 'update', получено '%s'", rootURLPath),
			statusCode: http.StatusBadRequest,
		}
		return
	}
	metricType, ok := metricsTypes[paramName]
	if ok {
		if metricType != typeName {
			err = &errorHTTP{
				message: fmt.Sprintf(
					"Некорректный тип для метрики. Ожидался '%s', получен '%s'", metricType, typeName),
				statusCode: http.StatusBadRequest,
			}
			return
		}
	} else {
		err = &errorHTTP{
			message:    fmt.Sprintf("Неизвестная метрика. Название полученной метрики '%s'", paramName),
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
	}
	if parseErr != nil {
		err = &errorHTTP{
			message: fmt.Sprintf(
				"Ошибка приведения значения '%s' метрики к типу '%s'", paramValueStr, typeName),
			statusCode: http.StatusBadRequest,
		}
		return
	}

	return &metric{typeName: typeName, paramName: paramName, paramValue: paramValue}, nil
}
