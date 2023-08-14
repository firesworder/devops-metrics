// Package server реализует серверную часть приложения(за исключением Storage).
// Содержит прежде всего хэндлеры и миддлвары, а также функциональность необходимую для работы сервера.
package server

import (
	"context"
	"database/sql"
	"errors"
	"github.com/firesworder/devopsmetrics/internal/filestore"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/server/env"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"log"
	"net"
	"time"
)

var (
	ErrNoDBConn = errors.New("no db connection")
)

// Server здесь хранится серверная логика(без привязки к HTTP/GRPC реализации)
type Server struct {
	env           *env.Environment
	fileStore     *filestore.FileStore
	writeTicker   *time.Ticker
	metricStorage storage.MetricRepository
	dbConn        *sql.DB
	trustedSubnet *net.IPNet
}

// NewServer конструктор для Server.
// Если перем-ая окружения DatabaseDsn установлена - использует ДБ для хранения метрик,
// иначе хранит в памяти + запись в файл.
func NewServer(e *env.Environment) (*Server, error) {
	server := Server{env: e}
	server.initFileStore()
	if server.env.DatabaseDsn == "" {
		server.initMetricStorage()
		server.initRepeatableSave()
	} else {
		sqlStorage, err := storage.NewSQLStorage(server.env.DatabaseDsn)
		if err != nil {
			return nil, err
		}
		server.metricStorage = sqlStorage
		server.dbConn = sqlStorage.Connection
	}

	if server.env.TrustedSubnet != "" {
		_, ipNet, err := net.ParseCIDR(server.env.TrustedSubnet)
		if err != nil {
			return nil, err
		}
		server.trustedSubnet = ipNet
	}

	return &server, nil
}

// todo: рефакторинг
// initFileStore инициализирует объект файл-хранилища метрик.
// Иниц-ия происходит только если DatabaseDsn не определен, а путь к файлу - определен.
func (s *Server) initFileStore() {
	if s.env.DatabaseDsn == "" && s.env.StoreFile != "" {
		s.fileStore = filestore.NewFileStore(s.env.StoreFile)
	}
}

// todo: рефакторинг ошибок
// initMetricStorage инициал-ет MetricStorage.
// Выполняется только при соблюдении условий.
func (s *Server) initMetricStorage() {
	if s.env.DatabaseDsn == "" && s.env.Restore && s.fileStore != nil {
		var err error
		s.metricStorage, err = s.fileStore.Read()
		if err != nil {
			log.Println(err)
			log.Println("Empty MemStorage was initialised")
			s.metricStorage = storage.NewMemStorage(map[string]storage.Metric{})
		}
		log.Println("MemStorage restored from store_file")
	} else {
		s.metricStorage = storage.NewMemStorage(map[string]storage.Metric{})
		log.Println("Empty MemStorage was initialised")
	}
}

// todo: рефакторинг
// initRepeatableSave регулярно(параметр StoreInterval) сохраняет состояние MetricStorage в файл.
// Выполняется только при соблюдении условий.
func (s *Server) initRepeatableSave() {
	if s.env.DatabaseDsn == "" && s.env.StoreInterval > 0 && s.fileStore != nil {
		go func() {
			var err error
			s.writeTicker = time.NewTicker(s.env.StoreInterval)
			for range s.writeTicker.C {
				// нет смысла писать nil MetricStorage
				if s.metricStorage == nil {
					continue
				}

				err = s.fileStore.Write(s.metricStorage)
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
	if s.env.DatabaseDsn == "" && s.env.StoreInterval == 0 && s.fileStore != nil && s.metricStorage != nil {
		err := s.fileStore.Write(s.metricStorage)
		return err
	}
	return nil
}

func (s *Server) GetAllMetrics(ctx context.Context) (map[string]storage.Metric, error) {
	return s.metricStorage.GetAll(ctx)
}

func (s *Server) Ping(ctx context.Context) error {
	if s.dbConn == nil {
		return ErrNoDBConn
	}
	return s.dbConn.Ping()
}

func (s *Server) UpdateMetric(ctx context.Context, metricMessage message.Metrics) (*message.Metrics, error) {
	var metric *storage.Metric
	var err error

	if err = message.CheckHash(metricMessage, s.env.Key); err != nil && !errors.Is(err, message.ErrEmptyKey) {
		return nil, err
	}

	metric, err = storage.NewMetricFromMessage(&metricMessage)
	if err != nil {
		return nil, err
	}

	err = s.metricStorage.UpdateOrAddMetric(ctx, *metric)
	if err != nil {
		return nil, err
	}

	if err = s.syncSaveMetricStorage(); err != nil {
		return nil, err
	}

	*metric, err = s.metricStorage.GetMetric(ctx, metric.Name)
	if err != nil {
		return nil, err
	}

	responseMsg := metric.GetMessageMetric()
	if err = responseMsg.InitHash(s.env.Key); err != nil && !errors.Is(err, message.ErrEmptyKey) {
		return nil, err
	}
	return &responseMsg, nil
}

func (s *Server) GetMetric(ctx context.Context, metricMessage message.Metrics) (*message.Metrics, error) {
	var err error
	metric, err := s.metricStorage.GetMetric(ctx, metricMessage.ID)
	if err != nil {
		return nil, err
	}

	metricMessage = metric.GetMessageMetric()
	if err = metricMessage.InitHash(s.env.Key); err != nil && !errors.Is(err, message.ErrEmptyKey) {
		return nil, err
	}
	return &metricMessage, nil
}

func (s *Server) BatchUpdate(ctx context.Context, metricMessagesBatch []message.Metrics) error {
	var metrics []storage.Metric

	for _, metricMessage := range metricMessagesBatch {
		var err error
		if err = message.CheckHash(metricMessage, s.env.Key); err != nil {
			return err
		}

		m, err := storage.NewMetricFromMessage(&metricMessage)
		if err != nil {
			return err
		}
		metrics = append(metrics, *m)
	}

	if err := s.metricStorage.BatchUpdate(ctx, metrics); err != nil {
		return err
	}

	if err := s.syncSaveMetricStorage(); err != nil {
		return err
	}
	return nil
}
