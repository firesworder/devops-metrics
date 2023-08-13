package env

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
	"time"
)

// todo: брать ключи из из jsonconfig констант
var testEnvVars = []string{
	"ADDRESS", "STORE_FILE", "STORE_INTERVAL", "RESTORE", "KEY", "DATABASE_DSN", "CRYPTO_KEY", "CONFIG", "TRUSTED_SUBNET",
}

func SaveOSVarsState(testEnvVars []string) map[string]string {
	osEnvVarsState := map[string]string{}
	for _, key := range testEnvVars {
		if v, ok := os.LookupEnv(key); ok {
			osEnvVarsState[key] = v
		}
	}
	return osEnvVarsState
}

func UpdateOSEnvState(t *testing.T, testEnvVars []string, newState map[string]string) {
	// удаляю переменные окружения, если они были до этого установлены
	for _, key := range testEnvVars {
		err := os.Unsetenv(key)
		require.NoError(t, err)
	}
	// устанавливаю переменные окружения использованные для теста
	for key, value := range newState {
		err := os.Setenv(key, value)
		require.NoError(t, err)
	}
}

func TestParseEnvArgs(t *testing.T) {
	// todo: сохранить Env и переменные окружения до изменения
	savedState := SaveOSVarsState(testEnvVars)
	envBefore := Env

	tests := []struct {
		name    string
		cmdStr  string
		envVars map[string]string
		wantEnv Environment
	}{
		{
			name:    "Test 1. Empty cmd args and env vars.",
			cmdStr:  "file.exe",
			envVars: map[string]string{},
			wantEnv: Environment{
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
		},

		// 2 server address
		{
			name:    "Test 2.1. 'ServerAddress'. Set by cmd",
			cmdStr:  "file.exe -a=cmd.site",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "cmd.site",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 2.2. 'ServerAddress'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"ADDRESS": "env.site"},
			wantEnv: Environment{
				ServerAddress:      "env.site",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},

		// 3 store file
		{
			name:    "Test 3.1. 'StoreFile'. Set by cmd",
			cmdStr:  "file.exe -f=/cmd/path",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/cmd/path",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 3.2. 'StoreFile'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"STORE_FILE": "/env/path"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/env/path",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},

		// 4 key
		{
			name:    "Test 4.1. 'Key'. Set by cmd",
			cmdStr:  "file.exe -k=ayaya",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "ayaya",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 4.2. 'Key'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"KEY": "tatata"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "tatata",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},

		// 5 DatabaseDsn
		{
			name:    "Test 5.1. 'DatabaseDsn'. Set by cmd",
			cmdStr:  "file.exe -d=postgres://cmd",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "postgres://cmd",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 5.2. 'DatabaseDsn'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"DATABASE_DSN": "postgres://env"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "postgres://env",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},

		// 6 Restore
		{
			name:    "Test 6.1. 'Restore'. Set by cmd",
			cmdStr:  "file.exe -r=false",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            false,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 6.2. 'Restore'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"RESTORE": "false"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            false,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},

		// 7 Store Interval
		{
			name:    "Test 7.1. 'StoreInterval'. Set by cmd",
			cmdStr:  "file.exe -i=20s",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      20 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 7.2. 'StoreInterval'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"STORE_INTERVAL": "50s"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      50 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},

		// 8 PrivateCryptoKeyFp
		{
			name:    "Test 8.1. 'PrivateCryptoKeyFp'. Set by cmd",
			cmdStr:  "file.exe -crypto-key=ayaya.fp",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "ayaya.fp",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 8.2. 'PrivateCryptoKeyFp'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"CRYPTO_KEY": "tatata.fp"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "tatata.fp",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			},
		},

		// 9 ConfigFilepath (file not exist!)
		{
			name:    "Test 9.1. 'ConfigFilepath'. Set by cmd, param 'c'",
			cmdStr:  "file.exe -c=/cmd/conf1.json",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "/cmd/conf1.json",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 9.2. 'ConfigFilepath'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"CONFIG": "/env/conf.json"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "/env/conf.json",
				TrustedSubnet:      "",
			},
		},
		{
			name:    "Test 9.3. 'ConfigFilepath'. Set by cmd, param 'config'",
			cmdStr:  "file.exe -config=/cmd/conf2.json",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "/cmd/conf2.json",
				TrustedSubnet:      "",
			},
		},

		// 10 TrustedSubnet
		{
			name:    "Test 10.1. 'TrustedSubnet'. Set by cmd",
			cmdStr:  "file.exe -t=192.168.1.1/24",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "192.168.1.1/24",
			},
		},
		{
			name:    "Test 10.2. 'TrustedSubnet'. Set by ENV",
			cmdStr:  "file.exe",
			envVars: map[string]string{"TRUSTED_SUBNET": "101.168.1.1/24"},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      300 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "101.168.1.1/24",
			},
		},

		// mixed cmd and env
		{
			name: "Test 11.1. All cmd and env params set",
			cmdStr: "file.exe -a=cmd.site -f=cmd.json -k=ayaya -d=postgres://cmd -r=false -i=20s " +
				"-crypto-key=/cmd/crypto -c=/cmd/conf1.json -config=/cmd/conf2.json -t=192.168.1.1/24",
			envVars: map[string]string{
				"ADDRESS":        "env.site",
				"STORE_FILE":     "env.json",
				"KEY":            "tatata",
				"DATABASE_DSN":   "postgres://env",
				"RESTORE":        "true",
				"STORE_INTERVAL": "60s",
				"CRYPTO_KEY":     "/env/crypto",
				"CONFIG":         "/env/conf.json",
				"TRUSTED_SUBNET": "101.168.1.1/24",
			},
			wantEnv: Environment{
				ServerAddress:      "env.site",
				StoreFile:          "env.json",
				Key:                "tatata",
				DatabaseDsn:        "postgres://env",
				Restore:            true,
				StoreInterval:      60 * time.Second,
				PrivateCryptoKeyFp: "/env/crypto",
				ConfigFilepath:     "/env/conf.json",
				TrustedSubnet:      "101.168.1.1/24",
			},
		},
		{
			name: "Test 11.2. All cmd params set",
			cmdStr: "file.exe -a=cmd.site -f=cmd.json -k=ayaya -d=postgres://cmd -r=false -i=20s " +
				"-crypto-key=/cmd/crypto -c=/cmd/conf1.json -config=/cmd/conf2.json -t=192.168.1.1/24",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "cmd.site",
				StoreFile:          "cmd.json",
				Key:                "ayaya",
				DatabaseDsn:        "postgres://cmd",
				Restore:            false,
				StoreInterval:      20 * time.Second,
				PrivateCryptoKeyFp: "/cmd/crypto",
				ConfigFilepath:     "/cmd/conf2.json",
				TrustedSubnet:      "192.168.1.1/24",
			},
		},

		{
			name:   "Test 11.3. All env params set",
			cmdStr: "file.exe",
			envVars: map[string]string{
				"ADDRESS":        "env.site",
				"STORE_FILE":     "env.json",
				"KEY":            "tatata",
				"DATABASE_DSN":   "postgres://env",
				"RESTORE":        "true",
				"STORE_INTERVAL": "60s",
				"CRYPTO_KEY":     "/env/crypto",
				"CONFIG":         "/env/conf.json",
				"TRUSTED_SUBNET": "101.168.1.1/24",
			},
			wantEnv: Environment{
				ServerAddress:      "env.site",
				StoreFile:          "env.json",
				Key:                "tatata",
				DatabaseDsn:        "postgres://env",
				Restore:            true,
				StoreInterval:      60 * time.Second,
				PrivateCryptoKeyFp: "/env/crypto",
				ConfigFilepath:     "/env/conf.json",
				TrustedSubnet:      "101.168.1.1/24",
			},
		},

		// json config
		{
			name: "Test 12.1. Cmd, ENV and config set",
			cmdStr: "file.exe -a=cmd.site -f=cmd.json -k=ayaya -d=postgres://cmd -r=false -i=20s " +
				"-crypto-key=/cmd/crypto -c=/cmd/conf1.json -config=/cmd/conf2.json -t=192.168.1.1/24",
			envVars: map[string]string{
				"ADDRESS":        "env.site",
				"STORE_FILE":     "env.json",
				"KEY":            "tatata",
				"DATABASE_DSN":   "postgres://env",
				"RESTORE":        "true",
				"STORE_INTERVAL": "60s",
				"CRYPTO_KEY":     "/env/crypto",
				"CONFIG":         "test_configs/all_fields_set.json",
				"TRUSTED_SUBNET": "101.168.1.1/24",
			},
			wantEnv: Environment{
				ServerAddress:      "env.site",
				StoreFile:          "env.json",
				Key:                "tatata",
				DatabaseDsn:        "postgres://env",
				Restore:            true,
				StoreInterval:      60 * time.Second,
				PrivateCryptoKeyFp: "/env/crypto",
				ConfigFilepath:     "test_configs/all_fields_set.json",
				TrustedSubnet:      "101.168.1.1/24",
			},
		},
		{
			name: "Test 12.2. Cmd and config set",
			cmdStr: "file.exe -a=cmd.site -f=cmd.json -k=ayaya -d=postgres://cmd -r=false -i=20s " +
				"-crypto-key=/cmd/crypto -c=/cmd/conf1.json -config=test_configs/all_fields_set.json -t=192.168.1.1/24",
			envVars: map[string]string{},
			wantEnv: Environment{
				ServerAddress:      "cmd.site",
				StoreFile:          "cmd.json",
				Key:                "ayaya",
				DatabaseDsn:        "postgres://cmd",
				Restore:            false,
				StoreInterval:      20 * time.Second,
				PrivateCryptoKeyFp: "/cmd/crypto",
				ConfigFilepath:     "test_configs/all_fields_set.json",
				TrustedSubnet:      "192.168.1.1/24",
			},
		},
		{
			name:   "Test 12.3. Only config set",
			cmdStr: "file.exe",
			envVars: map[string]string{
				"CONFIG": "test_configs/all_fields_set.json",
			},
			wantEnv: Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/path/to/file.db",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      1 * time.Second,
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "test_configs/all_fields_set.json",
				TrustedSubnet:      "192.168.1.1/24",
			},
		},
		{
			name:   "Test 12.4. Mixed cmd, ENV and config",
			cmdStr: "file.exe -a=cmd.site -d=postgres://cmd -config=test_configs/all_fields_set.json -t=192.168.1.1/24",
			envVars: map[string]string{
				"ADDRESS":        "env.site",
				"STORE_FILE":     "env.json",
				"RESTORE":        "true",
				"STORE_INTERVAL": "60s",
				"TRUSTED_SUBNET": "101.168.1.1/24",
			},
			wantEnv: Environment{
				ServerAddress:      "env.site",
				StoreFile:          "env.json",
				Key:                "",
				DatabaseDsn:        "postgres://cmd",
				Restore:            true,
				StoreInterval:      60 * time.Second,
				PrivateCryptoKeyFp: "/path/to/key.pem",
				ConfigFilepath:     "test_configs/all_fields_set.json",
				TrustedSubnet:      "101.168.1.1/24",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Устанавливаю env в дефолтные значения(обнулять я его не могу, т.к. flag линки потеряются?)
			Env = Environment{
				ServerAddress:      "localhost:8080",
				StoreFile:          "/tmp/devops-metrics-db.json",
				Key:                "",
				DatabaseDsn:        "",
				Restore:            true,
				StoreInterval:      1 * time.Second,
				PrivateCryptoKeyFp: "",
				ConfigFilepath:     "",
				TrustedSubnet:      "",
			}

			UpdateOSEnvState(t, testEnvVars, tt.envVars)
			// устанавливаю os.Args как эмулятор вызванной команды
			os.Args = strings.Split(tt.cmdStr, " ")
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.PanicOnError)
			initCmdArgs()

			// сама проверка корректности парсинга
			// todo: заменить паники на log.fatal
			require.NotPanics(t, ParseEnvArgs)
			assert.Equal(t, tt.wantEnv, Env)
		})
	}

	Env = envBefore
	UpdateOSEnvState(t, testEnvVars, savedState)
}
