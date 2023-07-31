// Package server реализует серверную часть приложения(за исключением Storage).
// Содержит прежде всего хэндлеры и миддлвары, а также функциональность необходимую для работы сервера.
package server

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"database/sql"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/crypt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v7"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/firesworder/devopsmetrics/internal/filestore"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
)

// Инициализирует параметры командной строки.
func init() {
	initCmdArgs()
}

// environment для получения(из ENV и cmd) и хранения переменных окружения агента.
type environment struct {
	ServerAddress      string        `env:"ADDRESS"`
	StoreFile          string        `env:"STORE_FILE"`
	Key                string        `env:"KEY"`
	DatabaseDsn        string        `env:"DATABASE_DSN"`
	Restore            bool          `env:"RESTORE"`
	StoreInterval      time.Duration `env:"STORE_INTERVAL"`
	PrivateCryptoKeyFp string        `env:"CRYPTO_KEY"`
	ConfigFilepath     string        `env:"CONFIG"`
}

// Env объект с переменными окружения(из ENV и cmd args).
var Env environment

// initCmdArgs Определяет флаги командной строки и линкует их с соотв полями объекта Env.
// В рамках этой же функции происходит и заполнение дефолтными значениями.
func initCmdArgs() {
	flag.StringVar(&Env.ServerAddress, "a", "localhost:8080", "server address")
	flag.BoolVar(&Env.Restore, "r", true, "restore memstorage from store file")
	flag.DurationVar(&Env.StoreInterval, "i", 300*time.Second, "store interval")
	flag.StringVar(&Env.StoreFile, "f", "/tmp/devops-metrics-db.json", "store file")
	flag.StringVar(&Env.Key, "k", "", "key for hash func")
	flag.StringVar(&Env.DatabaseDsn, "d", "", "database address")
	flag.StringVar(&Env.PrivateCryptoKeyFp, "crypto-key", "", "filepath to private key")
	flag.StringVar(&Env.ConfigFilepath, "config", "", "filepath to json env config")
	flag.StringVar(&Env.ConfigFilepath, "c", "", "filepath to json env config")
}

// ParseEnvArgs Парсит значения полей Env. Сначала из cmd аргументов, затем из перем-х окружения.
func ParseEnvArgs() {
	// Парсинг аргументов cmd
	flag.Parse()

	// Парсинг перем окружения
	err := env.Parse(&Env)
	if err != nil {
		log.Fatal(err)
	}

	// Парсинг json конфига
	err = parseJSONConfig()
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatal(err)
	}
}

// Server реализует серверную логику.
// Всё взаимодействие с серверной частью происходит через него.
type Server struct {
	FileStore     *filestore.FileStore
	WriteTicker   *time.Ticker
	Router        chi.Router
	MetricStorage storage.MetricRepository
	DBConn        *sql.DB
	LayoutsDir    string
	Decoder       *crypt.Decoder
}

// NewServer конструктор для Server.
// Если перем-ая окружения DatabaseDsn установлена - использует ДБ для хранения метрик,
// иначе хранит в памяти + запись в файл.
func NewServer() (*Server, error) {
	server := Server{}
	server.initFileStore()
	if Env.DatabaseDsn == "" {
		server.initMetricStorage()
		server.initRepeatableSave()
	} else {
		sqlStorage, err := storage.NewSQLStorage(Env.DatabaseDsn)
		if err != nil {
			return nil, err
		}
		server.MetricStorage = sqlStorage
		server.DBConn = sqlStorage.Connection
	}
	server.Router = server.newRouter()

	if Env.PrivateCryptoKeyFp != "" {
		decoder, err := crypt.NewDecoder(Env.PrivateCryptoKeyFp)
		if err != nil {
			return nil, err
		}
		server.Decoder = decoder
	}

	workingDir, _ := os.Getwd()
	server.LayoutsDir = filepath.Join(workingDir, "/internal/server/html_layouts")

	return &server, nil
}

// initFileStore инициализирует объект файл-хранилища метрик.
// Иниц-ия происходит только если DatabaseDsn не определен, а путь к файлу - определен.
func (s *Server) initFileStore() {
	if Env.DatabaseDsn == "" && Env.StoreFile != "" {
		s.FileStore = filestore.NewFileStore(Env.StoreFile)
	}
}

// initMetricStorage инициал-ет MetricStorage.
// Выполняется только при соблюдении условий.
func (s *Server) initMetricStorage() {
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

// initRepeatableSave регулярно(параметр StoreInterval) сохраняет состояние MetricStorage в файл.
// Выполняется только при соблюдении условий.
func (s *Server) initRepeatableSave() {
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

// syncSaveMetricStorage сохраняет MetricStorage в конце обработки успешного(200) запроса.
// Выполняется только при соблюдении условий.
func (s *Server) syncSaveMetricStorage() error {
	if Env.DatabaseDsn == "" && Env.StoreInterval == 0 && s.FileStore != nil && s.MetricStorage != nil {
		err := s.FileStore.Write(s.MetricStorage)
		return err
	}
	return nil
}

// newRouter определяет и возвращает роутер для сервера.
func (s *Server) newRouter() chi.Router {
	r := chi.NewRouter()

	r.Use(s.gzipDecompressor)
	r.Use(s.gzipCompressor)
	r.Use(s.decryptMessage)
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

// gzipResponseWriter для реализации gzipCompressor.
type gzipResponseWriter struct {
	http.ResponseWriter // нужен, чтобы хандлеры не спотыкались об отсутствие возм.установить header например.
	Writer              io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// gzipDecompressor - middleware для обработки входящих запросов с gzip сжатием.
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

// gzipDecompressor - middleware для gzip сжатия исходящих запросов.
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

// decryptMessage - middleware для расшифр. в асимм.шифровании
func (s *Server) decryptMessage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if s.Decoder != nil {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			r, err := s.Decoder.Decode(body)
			// если есть ошибка и это не ошибка расшифровки - выбросить http ошибку
			// если ошибка расшифровки - ничего не делать(оставить изначальный request.Body)
			if err != nil && !errors.Is(err, rsa.ErrDecryption) {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
				// если ошибок нет - заменить reader на расшифр.сообщение
			} else if err == nil {
				reader := io.NopCloser(bytes.NewReader(r))
				request.Body = reader
			}
		}
		next.ServeHTTP(writer, request)
	})
}

// Handlers

//	@Title			Server Devops API
//	@Description	Сервис сбора метрик.
//	@Version		1.0

//	@Contact.email	cus.never.again@yandex.ru

//	@BasePath	/
//	@Host		localhost:8080

//	@Tag.name			JSON
//	@Tag.description	"Группа JSON запросов."

//	@Tag.name			NoJSON
//	@Tag.description	"Группа запросов не использующих JSON."

// handlerShowAllMetrics godoc
//
//	@Tags		NoJSON
//	@Summary	Обрабатывает GET запросы вывода всех метрик сохраненных на сервере.
//	@ID			handlerShowAllMetrics
//	@Produce	html
//	@Success	200	{string}	string	"ok"
//	@Failure	500	{string}	string	"Внутренняя ошибка"
//	@Router		/ [get]
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
	if err = s.syncSaveMetricStorage(); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}
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
	if err = s.syncSaveMetricStorage(); err != nil {
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

// handlerPing godoc
//
//	@Tags		NoJSON
//	@Summary	Обрабатывает GET запрос доступности(ping) сервера.
//	@ID			handlerPing
//	@Success	200	{string}	string	"ok"
//	@Failure	500	{string}	string	"Внутренняя ошибка"	Ошибка	выдается,	если	БД	недоступна.
//	@Router		/ping [get]
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

	if err = s.syncSaveMetricStorage(); err != nil {
		http.Error(writer, err.Error(), http.StatusInternalServerError)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(http.StatusOK)
	// возвращаю пустую структуру, чтобы пройти автотест
	writer.Write([]byte("[]"))
}
