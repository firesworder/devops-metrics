package file_store

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"io"
	"os"
)

type FileStore struct {
	storeFilePath string
}

func (f *FileStore) Write(memStorage storage.MemStorage) error {
	file, err := os.OpenFile(f.storeFilePath, os.O_WRONLY|os.O_CREATE, 0644)
	defer file.Close()
	if err != nil {
		return err
	}

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

func (f *FileStore) Read() (*storage.MemStorage, error) {
	file, err := os.OpenFile(f.storeFilePath, os.O_RDONLY, 0644)
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
