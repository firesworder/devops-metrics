package env

import (
	"errors"
	"flag"
	"github.com/caarlos0/env/v7"
	"log"
	"os"
	"time"
)

// Инициализирует параметры командной строки.
func init() {
	initCmdArgs()
}

// Environment для получения(из ENV и cmd) и хранения переменных окружения агента.
type Environment struct {
	ServerAddress      string        `env:"ADDRESS"`
	StoreFile          string        `env:"STORE_FILE"`
	Key                string        `env:"KEY"`
	DatabaseDsn        string        `env:"DATABASE_DSN"`
	Restore            bool          `env:"RESTORE"`
	StoreInterval      time.Duration `env:"STORE_INTERVAL"`
	PrivateCryptoKeyFp string        `env:"CRYPTO_KEY"`
	ConfigFilepath     string        `env:"CONFIG"`
	TrustedSubnet      string        `env:"TRUSTED_SUBNET"`
}

// Env объект с переменными окружения(из ENV и cmd args).
var Env Environment

// initCmdArgs Определяет флаги командной строки и линкует их с соотв полями объекта Env.
// В рамках этой же функции происходит и заполнение дефолтными значениями.
func initCmdArgs() {
	flag.StringVar(&Env.ServerAddress, "a", "localhost:8080", "server address")
	flag.StringVar(&Env.StoreFile, "f", "/tmp/devops-metrics-db.json", "store file")
	flag.StringVar(&Env.Key, "k", "", "key for hash func")
	flag.StringVar(&Env.DatabaseDsn, "d", "", "database address")
	flag.BoolVar(&Env.Restore, "r", true, "restore memstorage from store file")
	flag.DurationVar(&Env.StoreInterval, "i", 300*time.Second, "store interval")
	flag.StringVar(&Env.PrivateCryptoKeyFp, "crypto-key", "", "filepath to private key")
	flag.StringVar(&Env.ConfigFilepath, "config", "", "filepath to json env config")
	flag.StringVar(&Env.ConfigFilepath, "c", "", "filepath to json env config")
	flag.StringVar(&Env.TrustedSubnet, "t", "", "trusted subnet")
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
