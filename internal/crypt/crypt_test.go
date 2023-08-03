package crypt

import (
	"encoding/json"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestNewEncoder(t *testing.T) {
	tests := []struct {
		name        string
		publicKeyFp string
		wantErr     bool
	}{
		{
			name:        "Test 1. Correct public key filepath.",
			publicKeyFp: "test/publicKey_1_test.pem",
			wantErr:     false,
		},
		{
			name:        "Test 2. Public key file is not exist",
			publicKeyFp: "test/file_not_exist.pem",
			wantErr:     true,
		},
		{
			name:        "Test 3. Public key file exist, but content is not public key",
			publicKeyFp: "test/privateKey_1_test.pem",
			wantErr:     true,
		},
		{
			name:        "Test 4. File exist, but empty",
			publicKeyFp: "test/empty_file.txt",
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEncoder(tt.publicKeyFp)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestNewDecoder(t *testing.T) {
	tests := []struct {
		name         string
		privateKeyFp string
		wantErr      bool
	}{
		{
			name:         "Test 1. Correct private key filepath.",
			privateKeyFp: "test/privateKey_1_test.pem",
			wantErr:      false,
		},
		{
			name:         "Test 2. Private key file is not exist",
			privateKeyFp: "test/file_not_exist.pem",
			wantErr:      true,
		},
		{
			name:         "Test 3. Private key file exist, but content is not private key",
			privateKeyFp: "test/publicKey_1_test.pem",
			wantErr:      true,
		},
		{
			name:         "Test 4. File exist, but empty",
			privateKeyFp: "test/empty_file.txt",
			wantErr:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewDecoder(tt.privateKeyFp)
			assert.Equal(t, tt.wantErr, err != nil)
		})
	}
}

func TestEncodeDecode(t *testing.T) {
	// подготовка сообщения
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

	// подготовка энкодеров и декодеров
	encoder1, err := NewEncoder("test/publicKey_1_test.pem")
	require.NoError(t, err)
	decoder1, err := NewDecoder("test/privateKey_1_test.pem")
	require.NoError(t, err)

	decoder2, err := NewDecoder("test/privateKey_2_test.pem")
	require.NoError(t, err)

	tests := []struct {
		name    string
		encoder *Encoder
		decoder *Decoder
		wantErr bool
	}{
		{
			name:    "Test 1. Correct encode->decode chain.",
			encoder: encoder1,
			decoder: decoder1,
			wantErr: false,
		},
		{
			name:    "Test 2. Incorrect pair cert+privateKey.",
			encoder: encoder1,
			decoder: decoder2,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var encodedMsg, gotMsg []byte
			encodedMsg, err = tt.encoder.Encode(msg)
			require.NoError(t, err)

			gotMsg, err = tt.decoder.Decode(encodedMsg)
			assert.Equal(t, tt.wantErr, err != nil)
			if err == nil {
				assert.Equal(t, string(msg), string(gotMsg))
			}
		})
	}
}
