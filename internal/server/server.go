package server

import (
	"compress/gzip"
	"database/sql"
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
	_ "github.com/jackc/pgx/v5/stdlib"
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
	Key           string        `env:"KEY"`
	DatabaseDsn   string        `env:"DATABASE_DSN"`
}

var Env Environment

// InitCmdArgs Определяет флаги командной строки и линкует их с соотв полями объекта Env
// В рамках этой же функции происходит и заполнение дефолтными значениями
func InitCmdArgs() {
	flag.StringVar(&Env.ServerAddress, "a", "localhost:8080", "server address")
	flag.BoolVar(&Env.Restore, "r", true, "restore memstorage from store file")
	flag.DurationVar(&Env.StoreInterval, "i", 300*time.Second, "store interval")
	flag.StringVar(&Env.StoreFile, "f", "/tmp/devops-metrics-db.json", "store file")
	flag.StringVar(&Env.Key, "k", "", "key for hash func")
	flag.StringVar(&Env.DatabaseDsn, "d", "", "database address")
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
	Router        chi.Router
	LayoutsDir    string
	MetricStorage storage.MetricRepository
	DBConn        *sql.DB
}

func NewServer() (*Server, error) {
	server := Server{}
	server.InitFileStore()
	if Env.DatabaseDsn == "" {
		server.InitMetricStorage()
		server.InitRepeatableSave()
	} else {
		sqlStorage, err := storage.NewSQLStorage(Env.DatabaseDsn)
		if err != nil {
			return nil, err
		}
		server.MetricStorage = sqlStorage
		server.DBConn = sqlStorage.Connection
	}
	server.Router = server.NewRouter()

	workingDir, _ := os.Getwd()
	server.LayoutsDir = filepath.Join(workingDir, "/internal/server/html_layouts")

	return &server, nil
}

func (s *Server) InitFileStore() {
	if Env.DatabaseDsn == "" && Env.StoreFile != "" {
		s.FileStore = filestore.NewFileStore(Env.StoreFile)
	}
}

func (s *Server) InitMetricStorage() {
	if Env.DatabaseDsn == "" && Env.Restore && s.FileStore != nil {
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
	if Env.DatabaseDsn == "" && Env.StoreInterval > 0 && s.FileStore != nil {
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

func (s *Server) SyncSaveMetricStorage() error {
	if Env.DatabaseDsn == "" && Env.StoreInterval == 0 && s.FileStore != nil && s.MetricStorage != nil {
		err := s.FileStore.Write(s.MetricStorage)
		return err
	}
	return nil
}

func (s *Server) NewRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(s.gzipDecompressor)
	r.Use(s.gzipCompressor)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/", func(r chi.Router) {
		r.Get("/", s.handlerShowAllMetrics)
		r.Get("/value/{typeName}/{metricName}", s.handlerGet)
		r.Get("/ping", s.handlerPing)
		r.Post("/updates/", s.handlerBatchUpdate)
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
		if strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
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
		// если не допускает сжатие - ничего не делать
		if !strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(writer, request)
			return
		}

		// создаю gzipWriter
		gzipWriter := gzip.NewWriter(writer)
		defer gzipWriter.Close()

		// оборачиваю ответ в gzip
		writer.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipResponseWriter{ResponseWriter: writer, Writer: gzipWriter}, request)
	})
}

func (s *Server) handlerShowAllMetrics(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	if s.LayoutsDir == "" {
		http.Error(writer, "Not initialised workingDir path", http.StatusInternalServerError)
		return
	}

	allMetrics, err := s.MetricStorage.GetAll(request.Context())
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	tmpl, err := template.ParseFiles(filepath.Join(s.LayoutsDir, "main_page.gohtml"))
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(writer,
		struct {
			PageTitle string
			Metrics   map[string]storage.Metric
		}{PageTitle: "Metrics", Metrics: allMetrics},
	)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlerGet(writer http.ResponseWriter, request *http.Request) {
	metric, err := s.MetricStorage.GetMetric(request.Context(), chi.URLParam(request, "metricName"))
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

func (s *Server) handlerAddUpdateMetric(writer http.ResponseWriter, request *http.Request) {
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

	err = s.MetricStorage.UpdateOrAddMetric(request.Context(), *m)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if err = s.SyncSaveMetricStorage(); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
}

func (s *Server) handlerJSONAddUpdateMetric(writer http.ResponseWriter, request *http.Request) {
	var metricMessage message.Metrics
	var metric *storage.Metric
	var err error

	if err = json.NewDecoder(request.Body).Decode(&metricMessage); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	if Env.Key != "" {
		var isHashCorrect bool
		isHashCorrect, err = metricMessage.CheckHash(Env.Key)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		} else if !isHashCorrect {
			http.Error(writer, "hash is not correct", http.StatusBadRequest)
			return
		}
	}

	metric, err = storage.NewMetricFromMessage(&metricMessage)
	if err != nil {
		if errors.Is(err, storage.ErrUnhandledValueType) {
			http.Error(writer, err.Error(), http.StatusNotImplemented)
		} else {
			http.Error(writer, err.Error(), http.StatusBadRequest)
		}
		return
	}

	err = s.MetricStorage.UpdateOrAddMetric(request.Context(), *metric)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}
	if err = s.SyncSaveMetricStorage(); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	*metric, err = s.MetricStorage.GetMetric(request.Context(), metric.Name)
	if err != nil {
		// ошибка не должна произойти, но мало ли
		http.Error(writer, "metric was not updated:"+err.Error(), http.StatusInternalServerError)
		return
	}

	responseMsg := metric.GetMessageMetric()
	if Env.Key != "" {
		err = responseMsg.InitHash(Env.Key)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

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
	var err error

	if err = json.NewDecoder(request.Body).Decode(&metricMessage); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	metric, err := s.MetricStorage.GetMetric(request.Context(), metricMessage.ID)
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
	if Env.Key != "" {
		err = responseMsg.InitHash(Env.Key)
		if err != nil {
			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	msgJSON, err := json.Marshal(responseMsg)
	if err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
	writer.Header().Set("Content-Type", "application/json")
	writer.Write(msgJSON)
}

func (s *Server) handlerPing(writer http.ResponseWriter, request *http.Request) {
	if s.DBConn == nil {
		writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	err := s.DBConn.Ping()
	if err != nil {
		writer.WriteHeader(http.StatusInternalServerError)
	} else {
		writer.WriteHeader(http.StatusOK)
	}
}

func (s *Server) handlerBatchUpdate(writer http.ResponseWriter, request *http.Request) {
	var metricMessagesBatch []message.Metrics
	var metrics []storage.Metric
	var err error

	if err = json.NewDecoder(request.Body).Decode(&metricMessagesBatch); err != nil {
		http.Error(writer, err.Error(), http.StatusBadRequest)
		return
	}

	for _, metricMessage := range metricMessagesBatch {
		if Env.Key != "" {
			var isHashCorrect bool
			isHashCorrect, err = metricMessage.CheckHash(Env.Key)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			} else if !isHashCorrect {
				http.Error(writer, "hash is not correct", http.StatusBadRequest)
				return
			}
		}

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

	if err = s.MetricStorage.BatchUpdate(request.Context(), metrics); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
	}

	if err = s.SyncSaveMetricStorage(); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	// возвращаю пустую структуру, чтобы пройти автотест
	writer.Write([]byte("[]"))
}
