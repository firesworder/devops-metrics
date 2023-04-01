package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/caarlos0/env/v7"
	"github.com/firesworder/devopsmetrics/internal/file_store"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Server struct {
	// todo: выделить ENV в ENV
	ServerAddress string        `env:"ADDRESS" envDefault:"localhost:8080"`
	StoreInterval time.Duration `env:"STORE_INTERVAL" envDefault:"300s"`
	StoreFile     string        `env:"STORE_FILE" envDefault:"/tmp/devops-metrics-db.json"`
	Restore       bool          `env:"RESTORE" envDefault:"true"`
	FileStore     *file_store.FileStore
	Router        chi.Router
	LayoutsDir    string
	MetricStorage storage.MetricRepository
}

func NewServer() *Server {
	server := Server{}
	server.initEnvParams()
	server.InitFileStore()
	server.InitMetricStorage()
	server.Router = server.NewRouter()

	workingDir, _ := os.Getwd()
	server.LayoutsDir = filepath.Join(workingDir, "/internal/server/html_layouts")

	return &server
}

func (s *Server) InitFileStore() {
	if s.StoreFile != "" {
		s.FileStore = file_store.NewFileStore(s.StoreFile)
	}
}

func (s *Server) InitMetricStorage() {
	if s.Restore && s.FileStore != nil {
		var err error
		s.MetricStorage, err = s.FileStore.Read()
		log.Println("MemStorage restored from store_file")
		if err != nil {
			log.Println(err)
		}
	}
	if !s.Restore || s.FileStore != nil {
		s.MetricStorage = storage.NewMemStorage(map[string]storage.Metric{})
		log.Println("Empty MemStorage was initialised")
	}
}

// todo: объединить взятие env
func (s *Server) initEnvParams() {
	err := env.Parse(s)
	if err != nil {
		panic(err)
	}
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
		r.Post("/update/", s.handlerJSONAddUpdateMetric)
		r.Post("/value/", s.handlerJSONGetMetric)
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
	if !ok {
		http.Error(writer, "unknown metric", http.StatusNotFound)
		return
	}
	writer.Write([]byte(metric.GetValueString()))
}

func (s *Server) handlerAddUpdateMetric(writer http.ResponseWriter, request *http.Request) {
	typeName := chi.URLParam(request, "typeName")
	metricName := chi.URLParam(request, "metricName")
	metricValue := chi.URLParam(request, "metricValue")

	m, metricError := storage.NewMetric(metricName, typeName, metricValue)
	if metricError != nil {
		if errors.Is(metricError, storage.ErrUnhandledValueType) {
			http.Error(writer, metricError.Error(), http.StatusNotImplemented)
		} else {
			http.Error(writer, metricError.Error(), http.StatusBadRequest)
		}
		return
	}

	errorObj := s.MetricStorage.UpdateOrAddMetric(*m)
	if errorObj != nil {
		http.Error(writer, errorObj.Error(), http.StatusBadRequest)
		return
	}
}

func (s *Server) handlerJSONAddUpdateMetric(writer http.ResponseWriter, request *http.Request) {
	var metricMessage message.Metrics

	if err := json.NewDecoder(request.Body).Decode(&metricMessage); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	m, metricError := storage.NewMetricFromMessage(&metricMessage)
	if metricError != nil {
		if errors.Is(metricError, storage.ErrUnhandledValueType) {
			http.Error(writer, metricError.Error(), http.StatusNotImplemented)
		} else {
			http.Error(writer, metricError.Error(), http.StatusBadRequest)
		}
		return
	}

	errorObj := s.MetricStorage.UpdateOrAddMetric(*m)
	if errorObj != nil {
		http.Error(writer, errorObj.Error(), http.StatusBadRequest)
		return
	}

	updatedMetric, ok := s.MetricStorage.GetMetric(m.Name)
	if !ok {
		// ошибка не должна произойти, но мало ли
		http.Error(writer, "metric was not updated", http.StatusInternalServerError)
		return
	}

	responseMsg := updatedMetric.GetMessageMetric()
	msgJSON, err := json.Marshal(responseMsg)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(msgJSON)
}

func (s *Server) handlerJSONGetMetric(writer http.ResponseWriter, request *http.Request) {
	var metricMessage message.Metrics

	if err := json.NewDecoder(request.Body).Decode(&metricMessage); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	metric, ok := s.MetricStorage.GetMetric(metricMessage.ID)
	if !ok {
		http.Error(
			writer,
			fmt.Sprintf("metric with name '%s' not found", metricMessage.ID),
			http.StatusNotFound,
		)
		return
	}

	responseMsg := metric.GetMessageMetric()
	msgJSON, err := json.Marshal(responseMsg)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(msgJSON)
}
