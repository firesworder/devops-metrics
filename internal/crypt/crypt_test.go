package crypt

import (
	"bytes"
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"testing"
)

func TestEncodeDecode(t *testing.T) {
	metricName, metricType, metricValue := "RandomValue", internal.GaugeTypeName, float64(12.13)
	metricMsg := message.Metrics{
		ID:    metricName,
		MType: metricType,
		Delta: nil,
		Value: &metricValue,
		Hash:  "",
	}
	err := metricMsg.InitHash("Ayayaka")
	require.NoError(t, err)
	msg, err := json.Marshal(metricMsg)
	require.NoError(t, err)

	tests := []struct {
		name          string
		msg           []byte
		certFp        string
		privateKeyFp  string
		wantEncodeErr bool
		wantDecodeErr bool
	}{
		{
			name:          "Test 1. Correct encode->decode chain.",
			msg:           msg,
			certFp:        "test\\publicKey_1_test.pem",
			privateKeyFp:  "test\\privateKey_1_test.pem",
			wantEncodeErr: false,
			wantDecodeErr: false,
		},
		{
			name:          "Test 2. Incorrect pair cert+privateKey.",
			msg:           msg,
			certFp:        "test\\publicKey_1_test.pem",
			privateKeyFp:  "test\\privateKey_2_test.pem",
			wantEncodeErr: false,
			wantDecodeErr: true,
		},
		{
			name:          "Test 3. Incorrect cert file.",
			msg:           msg,
			certFp:        "test\\privateKey_1_test.pem",
			privateKeyFp:  "test\\privateKey_2_test.pem",
			wantEncodeErr: true,
			wantDecodeErr: false,
		},
		{
			name:          "Test 4. Cert file is not exist.",
			msg:           msg,
			certFp:        "test\\publicKey_232323_test.pem",
			privateKeyFp:  "test\\privateKey_2_test.pem",
			wantEncodeErr: true,
			wantDecodeErr: false,
		},
		{
			name:          "Test 5. Incorrect privateKey file.",
			msg:           msg,
			certFp:        "test\\publicKey_1_test.pem",
			privateKeyFp:  "test\\publicKey_2_test.pem",
			wantEncodeErr: false,
			wantDecodeErr: true,
		},
		{
			name:          "Test 6. PrivateKey file is not exist.",
			msg:           msg,
			certFp:        "test\\publicKey_1_test.pem",
			privateKeyFp:  "test\\privateKey_33323_test.pem",
			wantEncodeErr: false,
			wantDecodeErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var encodedMsg, gotMsg []byte
			encodedMsg, err = Encode(tt.certFp, msg)
			require.Equal(t, tt.wantEncodeErr, err != nil)
			if err != nil {
				return
			}

			gotMsg, err = Decode(tt.privateKeyFp, encodedMsg)
			require.Equal(t, tt.wantDecodeErr, err != nil)
			if err != nil {
				return
			}

			assert.Equal(t, string(msg), string(gotMsg))
		})
	}
}

func TestNewReader(t *testing.T) {
	// шифруем сообщение ключом 1
	demoMsg := []byte("test message")
	encDemoMsg, err := Encode("test\\publicKey_1_test.pem", demoMsg)
	require.NoError(t, err)

	tests := []struct {
		name         string
		privateKeyFp string
		wantErr      bool
	}{
		{
			name:         "Test 1. Reader created successfully.",
			privateKeyFp: "test\\privateKey_1_test.pem",
			wantErr:      false,
		},
		{
			name:         "Test 2. Reader error(can not decrypt msg).",
			privateKeyFp: "test\\privateKey_2_test.pem",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqReader, err := http.NewRequest(http.MethodPost, "testUrl", bytes.NewReader(encDemoMsg))
			require.NoError(t, err)
			defer reqReader.Body.Close()

			r, err := NewReader(tt.privateKeyFp, reqReader.Body)
			reqReader.Body.Close()
			assert.Equal(t, tt.wantErr, err != nil)
			if err == nil {
				readerContent, err := io.ReadAll(r)
				require.NoError(t, err)
				defer r.Close()
				assert.Equal(t, readerContent, demoMsg)
			}
		})
	}
}
