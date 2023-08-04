package server

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/envconfighelper"
	"os"
	"time"
)

type envConfig struct {
	ServerAddress      string `json:"address"`
	Restore            bool   `json:"restore"`
	StoreInterval      string `json:"store_interval"`
	StoreFile          string `json:"store_file"`
	DatabaseDsn        string `json:"database_dsn"`
	PrivateCryptoKeyFp string `json:"crypto_key"`
	TrustedSubnet      string `json:"trusted_subnet"`
}

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

	// получаю json из конфига, путь беру из переменной env
	config, err := getJSONData(Env.ConfigFilepath)
	if err != nil {
		return err
	}

	// получаю список полей для заполнения из envJSON
	envconfighelper.GetFieldsNameToFill(cmdEnvDict, osEnvEnvDict, fieldsToSet)

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

func getJSONData(configFile string) (*envConfig, error) {
	config := &envConfig{}
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
