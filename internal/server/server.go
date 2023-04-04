package server

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/caarlos0/env/v7"
	"github.com/firesworder/devopsmetrics/internal/filestore"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func init() {
	InitCmdArgs()
}

type Environment struct {
	ServerAddress string        `env:"ADDRESS"`
	StoreInterval time.Duration `env:"STORE_INTERVAL"`
	StoreFile     string        `env:"STORE_FILE"`
	Restore       bool          `env:"RESTORE"`
}

var Env Environment

// InitCmdArgs Определяет флаги командной строки и линкует их с соотв полями объекта Env
// В рамках этой же функции происходит и заполнение дефолтными значениями
func InitCmdArgs() {
	flag.StringVar(&Env.ServerAddress, "a", "localhost:8080", "server address")
	flag.BoolVar(&Env.Restore, "r", true, "restore memstorage from store file")
	flag.DurationVar(&Env.StoreInterval, "i", 300*time.Second, "store interval")
	flag.StringVar(&Env.StoreFile, "f", "/tmp/devops-metrics-db.json", "store file")
}

// ParseEnvArgs Парсит значения полей Env. Сначала из cmd аргументов, затем из перем-х окружения
func ParseEnvArgs() {
	// Парсинг аргументов cmd
	flag.Parse()

	// Парсинг перем окружения
	err := env.Parse(&Env)
	if err != nil {
		panic(err)
	}
}

type Server struct {
	FileStore     *filestore.FileStore
	WriteTicker   *time.Ticker
	gzipWriter    *gzip.Writer
	Router        chi.Router
	LayoutsDir    string
	MetricStorage storage.MetricRepository
}

func NewServer() *Server {
	server := Server{}
	server.InitFileStore()
	server.InitMetricStorage()
	server.InitRepeatableSave()
	server.Router = server.NewRouter()

	workingDir, _ := os.Getwd()
	server.LayoutsDir = filepath.Join(workingDir, "/internal/server/html_layouts")

	return &server
}

func (s *Server) InitFileStore() {
	if Env.StoreFile != "" {
		s.FileStore = filestore.NewFileStore(Env.StoreFile)
	}
}

func (s *Server) InitMetricStorage() {
	if Env.Restore && s.FileStore != nil {
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
	if Env.StoreInterval > 0 && s.FileStore != nil {
		go func() {
			var err error
			s.WriteTicker = time.NewTicker(Env.StoreInterval)
			for range s.WriteTicker.C {
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

// todo: выглядит как неочень удобный костыль, переделать потом в рамках MemStorage
func (s *Server) SyncSaveMetricStorage() error {
	if Env.StoreInterval == 0 && s.FileStore != nil && s.MetricStorage != nil {
		err := s.FileStore.Write(s.MetricStorage)
		return err
	}
	return nil
}

func (s *Server) NewRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(s.gzipDecompressor)
	r.Use(s.gzipCompressor)
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

type gzipResponseWriter struct {
	http.ResponseWriter // нужен, чтобы хандлеры не спотыкались об отсутствие возм.установить header например.
	Writer              io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

func (s *Server) gzipDecompressor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Header.Get(`Content-Encoding`) == `gzip` {
			gz, err := gzip.NewReader(request.Body)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			request.Body = gz
			defer gz.Close()
		}
		next.ServeHTTP(writer, request)
	})
}

func (s *Server) gzipCompressor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var err error
		// если не допускает сжатие - ничего не делать
		if !strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(writer, request)
			return
		}

		// проверяет, существует уже writer gzip, если нет создает, иначе использует существующий
		if s.gzipWriter == nil {
			s.gzipWriter, err = gzip.NewWriterLevel(writer, gzip.BestSpeed)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
			}
		} else {
			s.gzipWriter.Reset(writer)
		}
		defer s.gzipWriter.Close()

		// оборачиваю ответ в gzip
		writer.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipResponseWriter{ResponseWriter: writer, Writer: s.gzipWriter}, request)
	})
}

func (s *Server) handlerShowAllMetrics(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
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
	writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
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
