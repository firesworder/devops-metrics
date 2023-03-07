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

func CustomHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return
	}

	urlParts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(urlParts) != 4 {
		errorMessage := fmt.Sprintf(
			"Некорректный URL запроса. Ожидаемое число частей пути URL: 4, получено %d", len(urlParts))
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	prefix, typeName, paramName, paramValueStr := urlParts[0], urlParts[1], urlParts[2], urlParts[3]
	if prefix != "update" {
		errorMessage := fmt.Sprintf(
			"Некорректный URL запроса. Ожидаемая первая часть пути 'update', получено '%s'", prefix)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}
	metricType, ok := metricsTypes[paramName]
	if ok {
		if metricType != typeName {
			errorMessage := fmt.Sprintf(
				"Некорректный тип для метрики. Ожидался '%s', получен '%s'", metricType, typeName)
			http.Error(w, errorMessage, http.StatusBadRequest)
			return
		}
	} else {
		errorMessage := fmt.Sprintf(
			"Неизвестная метрика. Название полученной метрики '%s'", paramName)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	var paramValue interface{}
	var err error
	switch typeName {
	case "counter":
		paramValue, err = strconv.ParseInt(paramValueStr, 10, 64)
	case "gauge":
		paramValue, err = strconv.ParseFloat(paramValueStr, 64)
	}
	if err != nil {
		errorMessage := fmt.Sprintf(
			"Ошибка приведения значения '%s' метрики к типу '%s'", paramValueStr, typeName)
		http.Error(w, errorMessage, http.StatusBadRequest)
		return
	}

	if typeName == "counter" {
		fmt.Printf("Type: %s | Param: %s | Value: %d\n", typeName, paramName, paramValue)
	} else if typeName == "gauge" {
		fmt.Printf("Type: %s | Param: %s | Value: %f\n", typeName, paramName, paramValue)
	}
}
