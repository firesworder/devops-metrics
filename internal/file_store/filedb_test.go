package file_store

import (
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

var metricCounter, _ = storage.NewMetric("PollCount", "counter", int64(10))
var metricGauge, _ = storage.NewMetric("RandomValue", "gauge", 12.133)

func init() {
	//ms := storage.NewMemStorage(map[string]storage.Metric{
	//	metricCounter.Name: *metricCounter,
	//	metricGauge.Name:   *metricGauge,
	//})
	//
	//jsonRes, err := json.Marshal(ms)
	//if err != nil {
	//	panic(err)
	//}
	//
	//err = os.WriteFile("files_test/read_correct_ms_test.json", jsonRes, 0644)
	//if err != nil {
	//	panic(err)
	//}
}

func TestFileStore_Read(t *testing.T) {
	tests := []struct {
		name          string
		storeFilePath string
		want          *storage.MemStorage
		wantErr       bool
	}{
		{
			name:          "Test #1. File db is not exist.",
			storeFilePath: "files_test/read_not_exist_test.json",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "Test #2. File db is empty.",
			storeFilePath: "files_test/read_empty_file_test.json",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "Test #3. File db content correct json memstorage object.",
			storeFilePath: "files_test/read_correct_ms_test.json",
			want: storage.NewMemStorage(map[string]storage.Metric{
				metricCounter.Name: *metricCounter,
				metricGauge.Name:   *metricGauge,
			}),
			wantErr: false,
		},
		{
			name:          "Test #4. File db content correct json, but not memstorage object.",
			storeFilePath: "files_test/not_ms.json",
			want:          nil,
			wantErr:       true,
		},
		{
			name:          "Test #5. File db content incorrect json.",
			storeFilePath: "files_test/incorrect_json.json",
			want:          nil,
			wantErr:       true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := FileStore{storeFilePath: tt.storeFilePath}
			got, err := f.Read()
			assert.Equal(t, tt.wantErr, err != nil)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFileStore_Write(t *testing.T) {
	tests := []struct {
		name          string
		ms            storage.MemStorage
		storeFilePath string
		wantContentAs string
		wantError     bool
	}{
		{
			name:          "Test #1. Empty storage. (new file)",
			ms:            *storage.NewMemStorage(map[string]storage.Metric{}),
			storeFilePath: "",
			wantContentAs: "files_test/read_empty_ms_test.json",
			wantError:     false,
		},
		{
			name: "Test #2. Filled storage. (new file)",
			ms: *storage.NewMemStorage(map[string]storage.Metric{
				metricCounter.Name: *metricCounter,
				metricGauge.Name:   *metricGauge,
			}),
			storeFilePath: "",
			wantContentAs: "files_test/read_correct_ms_test.json",
			wantError:     false,
		},
		{
			name: "Test #3. File with memstorage already exist.",
			ms: *storage.NewMemStorage(map[string]storage.Metric{
				metricCounter.Name: *metricCounter,
				metricGauge.Name:   *metricGauge,
			}),
			storeFilePath: "files_test/write_store_exist.json",
			wantContentAs: "files_test/read_correct_ms_test.json",
			wantError:     false,
		},
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var f FileStore
			if tt.storeFilePath != "" {
				f.storeFilePath = tt.storeFilePath
			} else {
				f.storeFilePath = fmt.Sprintf("files_test/write_test_%d.json", i)
				defer os.Remove(f.storeFilePath)
			}
			err := f.Write(tt.ms)
			assert.Equal(t, tt.wantError, err != nil)

			require.FileExists(t, f.storeFilePath)
			// todo: вынести получение файла в отдельную функцию
			wantContent, err := os.ReadFile(tt.wantContentAs)
			require.NoError(t, err)
			gotContent, err := os.ReadFile(f.storeFilePath)
			require.NoError(t, err)
			assert.Equal(t, wantContent, gotContent)
		})
	}
}
