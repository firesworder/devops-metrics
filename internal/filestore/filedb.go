// Package filestore используется в серверной части для организации хранения метрик в файле.
package filestore

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"

	"github.com/firesworder/devopsmetrics/internal/storage"
)

// FileStore реализует хранение файла метрик.
// Для использования доступны методы записи и чтения в\из файла storage.MetricRepository.
type FileStore struct {
	StoreFilePath string
}

// NewFileStore конструктор для FileStore.
// Если в storeFilePath передали пустую строку - возвращает nil, иначе FileStore объект.
func NewFileStore(storeFilePath string) *FileStore {
	// FileStore имеет смысл только с НЕ пустым путем к файлу
	if storeFilePath != "" {
		return &FileStore{StoreFilePath: storeFilePath}
	}
	return nil
}

// Write записывает объект storage.MetricRepository в файл.
func (f *FileStore) Write(memStorage storage.MetricRepository) error {
	if _, err := os.Stat(f.StoreFilePath); os.IsNotExist(err) {
		err = os.MkdirAll(filepath.Dir(f.StoreFilePath), 0644)
		if err != nil {
			return err
		}
	}
	file, err := os.OpenFile(f.StoreFilePath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	jsonMS, err := json.Marshal(&memStorage)
	if err != nil {
		return err
	}

	_, err = file.Write(jsonMS)
	if err != nil {
		return err
	}
	return nil
}

// Read читает объект storage.MetricRepository в файл.
// Если файла не существует - выбрасывает ошибку.
func (f *FileStore) Read() (*storage.MemStorage, error) {
	file, err := os.OpenFile(f.StoreFilePath, os.O_RDONLY, 0644)
	if err != nil {
		return nil, err
	}

	jsonMS, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	memStorage := storage.NewMemStorage(map[string]storage.Metric{})
	err = json.Unmarshal(jsonMS, &memStorage)
	if err != nil {
		return nil, err
	}
	return memStorage, nil
}
