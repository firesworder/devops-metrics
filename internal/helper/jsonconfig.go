package helper

import (
	"flag"
	"os"
)

// GetFieldsNameToFill обновляет fieldsToSet, отмечая false поля, которые были заполнены ДО парсинга конфига
func GetFieldsNameToFill(cmdDict, osEnvDict map[string]string, fieldsToSet map[string]bool) {
	// отмечаю поля заполненные через cmd параметры
	flag.Visit(func(f *flag.Flag) {
		if fieldName, ok := cmdDict[f.Name]; ok {
			fieldsToSet[fieldName] = false
		}
	})

	// отмечаю поля заполненные через ENV
	for envKey, fieldName := range osEnvDict {
		if _, ok := os.LookupEnv(envKey); ok {
			fieldsToSet[fieldName] = false
		}
	}
}
