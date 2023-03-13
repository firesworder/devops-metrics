package server

import (
	"fmt"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"html/template"
	"net/http"
	"path/filepath"
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
			statusCode: http.StatusNotFound,
		}
		return
	}
	rootURLPath, typeName, paramName, paramValueStr := urlParts[0], urlParts[1], urlParts[2], urlParts[3]
	if rootURLPath != mrh.rootURLPath {
		err = &errorHTTP{
			message: fmt.Sprintf(
				"Incorrect root part of URL. Expected '%s', got '%s'",
				mrh.rootURLPath, rootURLPath),
			statusCode: http.StatusNotFound,
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
		err = &errorHTTP{
			message:    "unhandled value type",
			statusCode: http.StatusNotImplemented,
		}
		return
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

type Server struct {
	Router        chi.Router
	MetricStorage storage.MetricRepository
}

func NewServer() *Server {
	server := Server{}
	server.Router = server.NewRouter()
	server.MetricStorage = storage.NewMemStorage(map[string]storage.Metric{})
	return &server
}

func (s *Server) NewRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/", func(r chi.Router) {
		r.Get("/", s.handlerRootPage)
		r.Get("/value/{typeName}/{metricName}", s.handlerGet)
		r.Post("/update/{typeName}/{metricName}/{metricValue}", s.handlerUpdate)
	})

	return r
}

func (s *Server) handlerRootPage(writer http.ResponseWriter, request *http.Request) {
	rootPathLayout := "./internal/server/html_layouts/main_page.gohtml"
	pageData := struct {
		PageTitle string
		Metrics   map[string]storage.Metric
	}{
		PageTitle: "Metrics",
		Metrics:   s.MetricStorage.GetAll(),
	}
	tmpl, err := template.ParseFiles(filepath.Join(internal.ProjectDir, rootPathLayout))
	if err != nil {
		fmt.Println(err)
		// todo: реализовать ошибку
		return
	}
	err = tmpl.Execute(writer, pageData)
	if err != nil {
		fmt.Println(err)
		// todo: реализовать ошибку
		return
	}
}

func (s *Server) handlerGet(writer http.ResponseWriter, request *http.Request) {
	_, metricName := chi.URLParam(request, "typeName"), chi.URLParam(request, "metricName")
	metric, ok := s.MetricStorage.GetMetric(metricName)
	if !ok {
		// todo: реализовать ошибку
		return
	}
	fmt.Fprintf(writer, "%v", metric.Value)
}

func (s *Server) handlerUpdate(writer http.ResponseWriter, request *http.Request) {
	typeName := chi.URLParam(request, "typeName")
	metricName := chi.URLParam(request, "metricName")
	metricValue := chi.URLParam(request, "metricValue")

	var paramValue interface{}
	var parseErr error
	// todo: убрать избыточный парсинг, перенести в metric
	switch typeName {
	case "counter":
		paramValue, parseErr = strconv.ParseInt(metricValue, 10, 64)
	case "gauge":
		paramValue, parseErr = strconv.ParseFloat(metricValue, 64)
	default:
		http.Error(writer, "unhandled value type", http.StatusNotImplemented)
		return
	}
	if parseErr != nil {
		http.Error(
			writer,
			fmt.Sprintf("Ошибка приведения значения '%s' метрики к типу '%s'", metricValue, typeName),
			http.StatusBadRequest,
		)
		return
	}

	m, metricError := storage.NewMetric(metricName, typeName, paramValue)
	if metricError != nil {
		http.Error(writer, metricError.Error(), http.StatusBadRequest)
		return
	}

	errorObj := s.MetricStorage.UpdateOrAddMetric(*m)
	if errorObj != nil {
		http.Error(writer, errorObj.Error(), http.StatusBadRequest)
		return
	}
}
