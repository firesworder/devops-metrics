package server

import (
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

type Server struct {
	Router        chi.Router
	LayoutsDir    string
	MetricStorage storage.MetricRepository
}

func NewServer() *Server {
	server := Server{}
	server.Router = server.NewRouter()

	workingDir, _ := os.Getwd()
	server.LayoutsDir = filepath.Join(workingDir, "/internal/server/html_layouts")

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
		r.Get("/", s.handlerShowAllMetrics)
		r.Get("/value/{typeName}/{metricName}", s.handlerGet)
		r.Post("/update/{typeName}/{metricName}/{metricValue}", s.handlerAddUpdateMetric)
	})

	return r
}

func (s *Server) handlerShowAllMetrics(writer http.ResponseWriter, request *http.Request) {
	if s.LayoutsDir == "" {
		http.Error(writer, "Not initialised workingDir path", http.StatusInternalServerError)
		return
	}
	pageData := struct {
		PageTitle string
		Metrics   map[string]storage.Metric
	}{
		PageTitle: "Metrics",
		Metrics:   s.MetricStorage.GetAll(),
	}

	tmpl, err := template.ParseFiles(filepath.Join(s.LayoutsDir, "main_page.gohtml"))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(writer, pageData)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlerGet(writer http.ResponseWriter, request *http.Request) {
	metricName := chi.URLParam(request, "metricName")
	metric, ok := s.MetricStorage.GetMetric(metricName)
	// todo: добавить проверку типа !ok || reflect.Type !=
	if !ok {
		http.Error(writer, "unknown metric", http.StatusNotFound)
		return
	}
	fmt.Fprintf(writer, "%v", metric.Value)
}

func (s *Server) handlerAddUpdateMetric(writer http.ResponseWriter, request *http.Request) {
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
			fmt.Sprintf("cannot cast metric value '%s' to type '%s'", metricValue, typeName),
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
