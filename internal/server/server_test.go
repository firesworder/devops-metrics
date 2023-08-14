package server

import (
	"github.com/firesworder/devopsmetrics/internal/filestore"
	"github.com/firesworder/devopsmetrics/internal/server/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	tests := []struct {
		name          string
		env           env.Environment
		wantFileStore bool
		wantRepSave   bool
		wantDBConn
		wantError bool
	}{
		{
			name: "Test 1. Empty(default) env",
			env: env.Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
			wantServer: &Server{
				env:           nil,
				fileStore:     filestore.NewFileStore("/tmp/devops-metrics-db.json"),
				writeTicker:   nil,
				metricStorage: nil,
				dbConn:        nil,
				trustedSubnet: nil,
			},
			wantError: false,
		},
		// с дев дсн(скип, если не подключается к дб)
		{
			name: "Test 2. Env with DatabaseDsn and TrustedSubnet",
			env: env.Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "postgresql://postgres:admin@localhost:5432/devops",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "192.168.1.1/24",
			},
			wantError: false,
		},
		{
			name: "Test 3. Env with DatabaseDsn error",
			env: env.Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "postgresql://invalid_dsn",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
			wantError: true,
		},
		{
			name: "Test 4. Env with TrustedSubnet error",
			env: env.Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "000.168.1/24",
			},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewServer(&tt.env)
			require.Equal(t, tt.wantError, err != nil)

			// проверка сервера
			if err != nil {
				assert.Equal(t, tt.wantServer, s)
			} else {
				assert.Equal(t, s.fileStore)
			}
		})
	}
}
