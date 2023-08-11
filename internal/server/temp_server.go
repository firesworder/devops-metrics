package server

import (
	"database/sql"
	"github.com/firesworder/devopsmetrics/internal/crypt"
	"github.com/firesworder/devopsmetrics/internal/filestore"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/go-chi/chi/v5"
	"log"
	"net"
	"os"
	"path/filepath"
	"time"
)

// TempServer здесь хранится серверная логика(без привязки к HTTP/GRPC реализации)
// todo: переименовать в Server
type TempServer struct {
	FileStore     *filestore.FileStore
	WriteTicker   *time.Ticker
	Router        chi.Router
	MetricStorage storage.MetricRepository
	DBConn        *sql.DB
	LayoutsDir    string
	Decoder       *crypt.Decoder
	TrustedSubnet *net.IPNet
}

// NewTempServer конструктор для Server.
// Если перем-ая окружения DatabaseDsn установлена - использует ДБ для хранения метрик,
// иначе хранит в памяти + запись в файл.
func NewTempServer() (*TempServer, error) {
	server := TempServer{}
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

	if Env.PrivateCryptoKeyFp != "" {
		decoder, err := crypt.NewDecoder(Env.PrivateCryptoKeyFp)
		if err != nil {
			return nil, err
		}
		server.Decoder = decoder
	}

	if Env.TrustedSubnet != "" {
		_, ipNet, err := net.ParseCIDR(Env.TrustedSubnet)
		if err != nil {
			return nil, err
		}
		server.TrustedSubnet = ipNet
	}

	workingDir, _ := os.Getwd()
	server.LayoutsDir = filepath.Join(workingDir, "/internal/server/html_layouts")

	return &server, nil
}

// initFileStore инициализирует объект файл-хранилища метрик.
// Иниц-ия происходит только если DatabaseDsn не определен, а путь к файлу - определен.
func (s *TempServer) initFileStore() {
	if Env.DatabaseDsn == "" && Env.StoreFile != "" {
		s.FileStore = filestore.NewFileStore(Env.StoreFile)
	}
}

// initMetricStorage инициал-ет MetricStorage.
// Выполняется только при соблюдении условий.
func (s *TempServer) initMetricStorage() {
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
func (s *TempServer) initRepeatableSave() {
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
func (s *TempServer) syncSaveMetricStorage() error {
	if Env.DatabaseDsn == "" && Env.StoreInterval == 0 && s.FileStore != nil && s.MetricStorage != nil {
		err := s.FileStore.Write(s.MetricStorage)
		return err
	}
	return nil
}
