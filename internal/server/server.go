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
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Server struct {
	// todo: выделить ENV в ENV
	ServerAddress string        `env:"ADDRESS" envDefault:"localhost:8080"`
	StoreInterval time.Duration `env:"STORE_INTERVAL" envDefault:"300s"`
	// Дефолтное значение для этого поля в initEnvParams
	StoreFile     string `env:"STORE_FILE"`
	Restore       bool   `env:"RESTORE" envDefault:"true"`
	FileStore     *file_store.FileStore
	WriteTicker   *time.Ticker
	Router        chi.Router
	LayoutsDir    string
	MetricStorage storage.MetricRepository
}

func NewServer() *Server {
	server := Server{}
	server.initEnvParams()
	server.InitFileStore()
	server.InitMetricStorage()
	server.InitRepeatableSave()
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
		if err != nil {
			log.Println(err)
			log.Println("Empty MemStorage was initialised")
			s.MetricStorage = storage.NewMemStorage(map[string]storage.Metric{})
		}
		log.Println("MemStorage restored from store_file")
	} else {
		s.MetricStorage = storage.NewMemStorage(map[string]storage.Metric{})
		log.Println("Empty MemStorage was initialised")
	}
}

func (s *Server) InitRepeatableSave() {
	if s.StoreInterval > 0 && s.FileStore != nil {
		go func() {
			var err error
			s.WriteTicker = time.NewTicker(s.StoreInterval)
			for _ = range s.WriteTicker.C {
				// нет смысла писать nil MetricStorage
				if s.MetricStorage == nil {
					continue
				}

				err = s.FileStore.Write(s.MetricStorage)
				if err != nil {
					log.Println(err)
				}
			}
		}()
	}
}

// todo: объединить взятие env
func (s *Server) initEnvParams() {
	err := env.Parse(s)
	if err != nil {
		panic(err)
	}

	// библиотека env не дает устанавливать значения "" и иметь envDefault тег одновременно
	path, isSet := os.LookupEnv("STORE_FILE")
	if isSet {
		s.StoreFile = path
	} else {
		s.StoreFile = "/tmp/devops-metrics-db.json"
	}
}

// todo: выглядит как неочень удобный костыль, переделать потом в рамках MemStorage
func (s *Server) SyncSaveMetricStorage() error {
	if s.StoreInterval == 0 && s.FileStore != nil && s.MetricStorage != nil {
		err := s.FileStore.Write(s.MetricStorage)
		return err
	}
	return nil
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
	if errorObj = s.SyncSaveMetricStorage(); errorObj != nil {
		log.Println(errorObj)
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
	if errorObj = s.SyncSaveMetricStorage(); errorObj != nil {
		log.Println(errorObj)
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
