package env

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/helper"
	"os"
	"time"
)

type serverEnvConfig struct {
	ServerAddress      string `json:"address"`
	Restore            bool   `json:"restore"`
	StoreInterval      string `json:"store_interval"`
	StoreFile          string `json:"store_file"`
	DatabaseDsn        string `json:"database_dsn"`
	PrivateCryptoKeyFp string `json:"crypto_key"`
	TrustedSubnet      string `json:"trusted_subnet"`
}

// todo: можно упростить эти константы?

// словарь [ключ ком.строки: имя ассоц. поля Env]
var cmdEnvDict = map[string]string{
	"a":          "ServerAddress",
	"r":          "Restore",
	"i":          "StoreInterval",
	"f":          "StoreFile",
	"d":          "DatabaseDsn",
	"crypto-key": "PrivateCryptoKeyFp",
	"t":          "TrustedSubnet",
}

// словарь [перем.окружения: имя ассоц. поля Env]
var osEnvEnvDict = map[string]string{
	"ADDRESS":        "ServerAddress",
	"RESTORE":        "Restore",
	"STORE_INTERVAL": "StoreInterval",
	"STORE_FILE":     "StoreFile",
	"DATABASE_DSN":   "DatabaseDsn",
	"CRYPTO_KEY":     "PrivateCryptoKeyFp",
	"TRUSTED_SUBNET": "TrustedSubnet",
}

// todo: вынести путь к конфигу в аргументы
func parseJSONConfig() error {
	// поля заполняемые из JSON(константа)
	var fieldsToSet = map[string]bool{
		"ServerAddress":      true,
		"Restore":            true,
		"StoreInterval":      true,
		"StoreFile":          true,
		"DatabaseDsn":        true,
		"PrivateCryptoKeyFp": true,
		"TrustedSubnet":      true,
	}

	// получаю json из конфига, путь беру из переменной env
	config, err := getJSONData(Env.ConfigFilepath)
	if err != nil {
		return err
	}

	// получаю список полей для заполнения из envJSON
	helper.GetFieldsNameToFill(cmdEnvDict, osEnvEnvDict, fieldsToSet)

	// записываю новые значения
	if fieldsToSet["ServerAddress"] {
		Env.ServerAddress = config.ServerAddress
	}
	if fieldsToSet["Restore"] {
		Env.Restore = config.Restore
	}
	if fieldsToSet["StoreInterval"] {
		dur, err := time.ParseDuration(config.StoreInterval)
		if err != nil {
			return err
		}
		Env.StoreInterval = dur
	}
	if fieldsToSet["StoreFile"] {
		Env.StoreFile = config.StoreFile
	}
	if fieldsToSet["DatabaseDsn"] {
		Env.DatabaseDsn = config.DatabaseDsn
	}
	if fieldsToSet["PrivateCryptoKeyFp"] {
		Env.PrivateCryptoKeyFp = config.PrivateCryptoKeyFp
	}
	if fieldsToSet["TrustedSubnet"] {
		Env.TrustedSubnet = config.TrustedSubnet
	}
	return nil
}

func getJSONData(configFile string) (*serverEnvConfig, error) {
	config := &serverEnvConfig{}
	f, err := os.Open(configFile)
	if err != nil {
		return nil, err
	}
	err = json.NewDecoder(f).Decode(config)
	if err != nil {
		return nil, err
	}
	return config, nil
}
