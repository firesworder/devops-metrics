package agent

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/envconfighelper"
	"os"
	"time"
)

type envConfig struct {
	ServerAddress     string `json:"address"`
	ReportInterval    string `json:"report_interval"`
	PollInterval      string `json:"poll_interval"`
	PublicCryptoKeyFp string `json:"crypto_key"`
}

func parseJSONConfig() error {
	// поля заполняемые из JSON(константа)
	var fieldsToSet = map[string]bool{
		"ServerAddress":  true,
		"ReportInterval": true,
		"PollInterval":   true,
		"CryptoKey":      true,
	}

	// словарь [ключ ком.строки: имя ассоц. поля Env]
	var cmdEnvDict = map[string]string{
		"a":          "ServerAddress",
		"r":          "ReportInterval",
		"p":          "PollInterval",
		"crypto-key": "CryptoKey",
	}

	// словарь [перем.окружения: имя ассоц. поля Env]
	var osEnvEnvDict = map[string]string{
		"ADDRESS":         "ServerAddress",
		"REPORT_INTERVAL": "ReportInterval",
		"POLL_INTERVAL":   "PollInterval",
		"CRYPTO_KEY":      "CryptoKey",
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
	if fieldsToSet["ReportInterval"] {
		dur, err := time.ParseDuration(config.ReportInterval)
		if err != nil {
			return err
		}
		Env.ReportInterval = dur
	}
	if fieldsToSet["PollInterval"] {
		dur, err := time.ParseDuration(config.PollInterval)
		if err != nil {
			return err
		}
		Env.PollInterval = dur
	}
	if fieldsToSet["CryptoKey"] {
		Env.PublicCryptoKeyFp = config.PublicCryptoKeyFp
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
