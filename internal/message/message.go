package message

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"github.com/firesworder/devopsmetrics/internal"
)

type Metrics struct {
	ID    string   `json:"id"`              // Имя метрики
	MType string   `json:"type"`            // Параметр, принимающий значение gauge или counter
	Delta *int64   `json:"delta,omitempty"` // Значение метрики в случае передачи counter
	Value *float64 `json:"value,omitempty"` // Значение метрики в случае передачи gauge
	Hash  string   `json:"hash,omitempty"`  // Значение хеш-функции
}

func (m *Metrics) InitHash(key string) error {
	if key == "" {
		return fmt.Errorf("key cannot be empty")
	}

	h := hmac.New(sha256.New, []byte(key))
	switch m.MType {
	case internal.GaugeTypeName:
		if m.Value == nil {
			return fmt.Errorf("value cannot be nil for type gauge")
		}
		h.Write([]byte(fmt.Sprintf("%s:gauge:%f", m.ID, *m.Value)))
	case internal.CounterTypeName:
		if m.Delta == nil {
			return fmt.Errorf("delta cannot be nil for type counter")
		}
		h.Write([]byte(fmt.Sprintf("%s:counter:%d", m.ID, *m.Delta)))
	default:
		return fmt.Errorf("unhandled type '%s'", m.MType)
	}
	m.Hash = hex.EncodeToString(h.Sum(nil))

	return nil
}

func (m *Metrics) CheckHash(key string) (bool, error) {
	gotHash := m.Hash
	defer func() {
		m.Hash = gotHash
	}()

	err := m.InitHash(key)
	if err != nil {
		return false, err
	}
	return hmac.Equal([]byte(gotHash), []byte(m.Hash)), nil
}
