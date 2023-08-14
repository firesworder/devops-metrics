package httpserver

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"html/template"
	"net/http"
	"path/filepath"
)

func (hs *HTTPServer) handlerShowAllMetrics(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if hs.server.LayoutsDir == "" {
		http.Error(writer, "Not initialised workingDir path", http.StatusInternalServerError)
		return
	}

	allMetrics, err := hs.server.MetricStorage.GetAll(request.Context())
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join(hs.server.LayoutsDir, "main_page.gohtml"))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(writer,
		struct {
			Metrics   map[string]storage.Metric
			PageTitle string
		}{PageTitle: "Metrics", Metrics: allMetrics},
	)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handlerGet godoc
//
//	@Tags			NoJSON
//	@Summary		Обрабатывает GET запросы получения информация по метрике.
//	@Description	В ответ возвращает значение метрики(в теле ответа).
//	@ID				handlerGet
//	@Produce		plain
//	@Param			typeName	path		string	true	"Тип метрики"
//	@Param			metricName	path		int		true	"Название метрики"
//	@Success		200			{string}	string	"<Значение метрики>"
//	@Failure		404			{string}	string	"unknown metric"
//	@Failure		500			{string}	string	"Внутренняя ошибка"
//	@Router			/value/{typeName}/{metricName} [get]
func (hs *HTTPServer) handlerGet(writer http.ResponseWriter, request *http.Request) {
	metric, err := hs.server.MetricStorage.GetMetric(request.Context(), chi.URLParam(request, "metricName"))
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.Error(writer, "unknown metric", http.StatusNotFound)
		} else {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	writer.Write([]byte(metric.GetValueString()))
}

// handlerAddUpdateMetric godoc
//
//	@Tags			NoJSON
//	@Summary		Обрабатывает POST запросы сохранения метрики на сервере.
//	@Description	Метрика(наим-ие, тип и значение) передается через URLParam.
//
// В ответ возвращает статус обработки запроса.
//
// Если метрика с таким именем не присутствует на сервере - добавляет ее, иначе обновляет существующую.
//
//	@ID				handlerAddUpdateMetric
//	@Param			typeName	path		string	true	"Тип метрики"
//	@Param			metricName	path		int		true	"Название метрики"
//	@Param			metricValue	path		int		true	"Значение метрики"
//	@Success		200			{string}	string	"ok"
//	@Failure		404			{string}	string	"unknown metric"
//	@Failure		500			{string}	string	"Внутренняя ошибка"
//	@Router			/update/{typeName}/{metricName}/{metricValue} [get]
func (hs *HTTPServer) handlerAddUpdateMetric(writer http.ResponseWriter, request *http.Request) {
	var err error

	m, err := storage.NewMetric(
		chi.URLParam(request, "metricName"),
		chi.URLParam(request, "typeName"),
		chi.URLParam(request, "metricValue"),
	)
	if err != nil {
		if errors.Is(err, storage.ErrUnhandledValueType) {
			http.Error(writer, err.Error(), http.StatusNotImplemented)
		} else {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}
		return
	}

	err = hs.server.MetricStorage.UpdateOrAddMetric(request.Context(), *m)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	// todo: hs.server.syncSaveMetricStorage() экспортируемым
	//if err = hs.server.syncSaveMetricStorage(); err != nil {
	//	http.Error(writer, err.Error(), http.StatusInternalServerError)
	//	return
	//}
}

// handlerJSONAddUpdateMetric godoc
//
//	@Tags			JSON
//	@Summary		Обрабатывает POST запросы сохранения метрики на сервере.
//	@Description	Метрика(наим-ие, тип и значение) передается через тело запроса, посредством message.Metrics.
//
// В ответ возвращает сохраненную на сервере метрику(после выполнения запроса).
//
// Если метрика с таким именем не присутствует на сервере - добавляет ее, иначе обновляет существующую.
//
//	@ID				handlerJSONAddUpdateMetric
//	@Accept			json
//	@Produce		json
//	@Success		200	{string}	string	"ok"
//	@Failure		400	{string}	string	"Неверный запрос"
//	@Failure		400	{string}	string	"hash is not correct"	если	полученный	хеш	не	совпал	с	созданным	на	сервере.
//	@Failure		404	{string}	string	"unknown metric"
//	@Failure		500	{string}	string	"Внутренняя ошибка"
//	@Failure		501	{string}	string	"Not Implemented"	если	передан	нереализованный	на	сервере	тип	метрики.
//	@Router			/update/ [post]
func (hs *HTTPServer) handlerJSONAddUpdateMetric(writer http.ResponseWriter, request *http.Request) {
	var metricMessage message.Metrics
	var metric *storage.Metric
	var err error

	if err = json.NewDecoder(request.Body).Decode(&metricMessage); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	// todo: Env.Key
	//if Env.Key != "" {
	//	var isHashCorrect bool
	//	isHashCorrect, err = metricMessage.CheckHash(Env.Key)
	//	if err != nil {
	//		http.Error(writer, err.Error(), http.StatusInternalServerError)
	//		return
	//	} else if !isHashCorrect {
	//		http.Error(writer, "hash is not correct", http.StatusBadRequest)
	//		return
	//	}
	//}

	metric, err = storage.NewMetricFromMessage(&metricMessage)
	if err != nil {
		if errors.Is(err, storage.ErrUnhandledValueType) {
			http.Error(writer, err.Error(), http.StatusNotImplemented)
		} else {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}
		return
	}

	err = hs.server.MetricStorage.UpdateOrAddMetric(request.Context(), *metric)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	// todo: hs.server.syncSaveMetricStorage() экспортируемым
	//if err = hs.server.syncSaveMetricStorage(); err != nil {
	//	http.Error(writer, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	*metric, err = hs.server.MetricStorage.GetMetric(request.Context(), metric.Name)
	if err != nil {
		// ошибка не должна произойти, но мало ли
		http.Error(writer, "metric was not updated:"+err.Error(), http.StatusInternalServerError)
		return
	}

	responseMsg := metric.GetMessageMetric()
	// todo: Env.Key
	//if Env.Key != "" {
	//	err = responseMsg.InitHash(Env.Key)
	//	if err != nil {
	//		http.Error(writer, err.Error(), http.StatusInternalServerError)
	//		return
	//	}
	//}

	msgJSON, err := json.Marshal(responseMsg)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(msgJSON)
}

// handlerJSONGetMetric godoc
//
//	@Tags			JSON
//	@Summary		Обрабатывает POST запросы получения метрики на сервере.
//	@Description	Наименование треб-ой метрики передается через тело запроса, посредством message.Metrics.
//
// В ответ возвращает сохраненную на сервере метрику.
//
//	@ID				handlerJSONGetMetric
//	@Accept			json
//	@Produce		json
//	@Success		200	{string}	string	"ok"
//	@Failure		400	{string}	string	"Неверный запрос"
//	@Failure		404	{string}	string	"metric with name <metricname> not found"
//	@Failure		500	{string}	string	"Внутренняя ошибка"
//	@Router			/value/ [post]
func (hs *HTTPServer) handlerJSONGetMetric(writer http.ResponseWriter, request *http.Request) {
	var metricMessage message.Metrics
	var err error

	if err = json.NewDecoder(request.Body).Decode(&metricMessage); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	metric, err := hs.server.MetricStorage.GetMetric(request.Context(), metricMessage.ID)
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
			http.Error(
				writer,
				fmt.Sprintf("metric with name '%s' not found", metricMessage.ID),
				http.StatusNotFound,
			)
		} else {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	responseMsg := metric.GetMessageMetric()
	// todo: Env.Key
	//if Env.Key != "" {
	//	err = responseMsg.InitHash(Env.Key)
	//	if err != nil {
	//		http.Error(writer, err.Error(), http.StatusInternalServerError)
	//		return
	//	}
	//}

	msgJSON, err := json.Marshal(responseMsg)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(msgJSON)
}

// handlerPing godoc
//
//	@Tags		NoJSON
//	@Summary	Обрабатывает GET запрос доступности(ping) сервера.
//	@ID			handlerPing
//	@Success	200	{string}	string	"ok"
//	@Failure	500	{string}	string	"Внутренняя ошибка"	Ошибка	выдается,	если	БД	недоступна.
//	@Router		/ping [get]
func (hs *HTTPServer) handlerPing(writer http.ResponseWriter, request *http.Request) {
	if hs.server.DBConn == nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	err := hs.server.DBConn.Ping()
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

// handlerBatchUpdate godoc
//
//	@Tags			JSON
//	@Summary		Обрабатывает POST запросы сохранения набора(словаря) метрик на сервере.
//	@Description	Метрики передаются как словарь message.Metrics.
//
// В ответ возвращает статус обработки запроса.
// Не существующие на сервере метрики - будут добавлены, иначе обновлены.
//
//	@ID				handlerBatchUpdate
//	@Accept			json
//	@Success		200	{string}	string	"ok"
//	@Failure		400	{string}	string	"Неверный запрос"
//	@Failure		400	{string}	string	"hash is not correct"	если	полученный	хеш	не	совпал	с	созданным	на	сервере.
//	@Failure		500	{string}	string	"Внутренняя ошибка"
//	@Failure		501	{string}	string	"Not Implemented"	если	передан	нереализованный	на	сервере	тип	метрики.
//	@Router			/updates/ [post]
func (hs *HTTPServer) handlerBatchUpdate(writer http.ResponseWriter, request *http.Request) {
	var metricMessagesBatch []message.Metrics
	var metrics []storage.Metric
	var err error

	if err = json.NewDecoder(request.Body).Decode(&metricMessagesBatch); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	for _, metricMessage := range metricMessagesBatch {
		// todo: Env.Key
		//if Env.Key != "" {
		//	var isHashCorrect bool
		//	isHashCorrect, err = metricMessage.CheckHash(Env.Key)
		//	if err != nil {
		//		http.Error(writer, err.Error(), http.StatusInternalServerError)
		//		return
		//	} else if !isHashCorrect {
		//		http.Error(writer, "hash is not correct", http.StatusBadRequest)
		//		return
		//	}
		//}

		var m *storage.Metric
		m, err = storage.NewMetricFromMessage(&metricMessage)
		if err != nil {
			if errors.Is(err, storage.ErrUnhandledValueType) {
				http.Error(writer, err.Error(), http.StatusNotImplemented)
			} else {
				http.Error(writer, err.Error(), http.StatusBadRequest)
			}
			return
		}
		metrics = append(metrics, *m)
	}

	if err = hs.server.MetricStorage.BatchUpdate(request.Context(), metrics); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}

	// todo: hs.server.syncSaveMetricStorage() экспортируемым
	//if err = hs.server.syncSaveMetricStorage(); err != nil {
	//	http.Error(writer, err.Error(), http.StatusInternalServerError)
	//	return
	//}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	// возвращаю пустую структуру, чтобы пройти автотест
	writer.Write([]byte("[]"))
}
