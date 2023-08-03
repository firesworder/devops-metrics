package envconfighelper

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"strings"
	"testing"
)

var testEnvVars = []string{"ADDRESS", "REPORT_INTERVAL", "POLL_INTERVAL"}

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

func TestGetFieldsNameToFill(t *testing.T) {
	var serverAddressFlag string

	savedState := SaveOSVarsState(testEnvVars)
	tests := []struct {
		name            string
		cmdDict         map[string]string
		osEnvDict       map[string]string
		fieldsToSet     map[string]bool
		cmdStr          string
		envVars         map[string]string
		wantFieldsToSet map[string]bool
	}{
		{
			name: "Test 1. All fields to false",
			cmdDict: map[string]string{
				"a": "ServerAddress",
			},
			osEnvDict: map[string]string{
				"REPORT_INTERVAL": "ReportInterval",
				"POLL_INTERVAL":   "PollInterval",
			},
			fieldsToSet: map[string]bool{
				"ServerAddress":  true,
				"ReportInterval": true,
				"PollInterval":   true,
			},
			cmdStr: "file.exe -a=site.com",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "POLL_INTERVAL": "5s",
			},
			wantFieldsToSet: map[string]bool{
				"ServerAddress":  false,
				"ReportInterval": false,
				"PollInterval":   false,
			},
		},
		{
			name: "Test 2. PollInterval(by env) is true",
			cmdDict: map[string]string{
				"a": "ServerAddress",
			},
			osEnvDict: map[string]string{
				"REPORT_INTERVAL": "ReportInterval",
				"POLL_INTERVAL":   "PollInterval",
			},
			fieldsToSet: map[string]bool{
				"ServerAddress":  true,
				"ReportInterval": true,
				"PollInterval":   true,
			},
			cmdStr: "file.exe -a=site.com",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "KEY": "5s",
			},
			wantFieldsToSet: map[string]bool{
				"ServerAddress":  false,
				"ReportInterval": false,
				"PollInterval":   true,
			},
		},
		{
			name: "Test 3. ServerAddress(by cmd) is true",
			cmdDict: map[string]string{
				"a": "ServerAddress",
			},
			osEnvDict: map[string]string{
				"REPORT_INTERVAL": "ReportInterval",
				"POLL_INTERVAL":   "PollInterval",
			},
			fieldsToSet: map[string]bool{
				"ServerAddress":  true,
				"ReportInterval": true,
				"PollInterval":   true,
			},
			cmdStr: "file.exe",
			envVars: map[string]string{
				"REPORT_INTERVAL": "20s", "KEY": "5s", "POLL_INTERVAL": "5s",
			},
			wantFieldsToSet: map[string]bool{
				"ServerAddress":  true,
				"ReportInterval": false,
				"PollInterval":   false,
			},
		},
		{
			name: "Test 4. All fiedls is true",
			cmdDict: map[string]string{
				"a": "ServerAddress",
			},
			osEnvDict: map[string]string{
				"REPORT_INTERVAL": "ReportInterval",
				"POLL_INTERVAL":   "PollInterval",
			},
			fieldsToSet: map[string]bool{
				"ServerAddress":  true,
				"ReportInterval": true,
				"PollInterval":   true,
			},
			cmdStr: "file.exe",
			envVars: map[string]string{
				"SOME_VAR1": "20s",
			},
			wantFieldsToSet: map[string]bool{
				"ServerAddress":  true,
				"ReportInterval": true,
				"PollInterval":   true,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Args = strings.Split(tt.cmdStr, " ")
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.PanicOnError)
			flag.StringVar(&serverAddressFlag, "a", "some_site.com", "")
			flag.Parse()

			UpdateOSEnvState(t, testEnvVars, tt.envVars)

			GetFieldsNameToFill(tt.cmdDict, tt.osEnvDict, tt.fieldsToSet)
			assert.Equal(t, tt.fieldsToSet, tt.wantFieldsToSet)
		})
	}
	UpdateOSEnvState(t, testEnvVars, savedState)
}
