package crypt

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

// Encoder RSA энкодер, при помощи прив.ключа шифрует сообщения.
type Encoder struct {
	publicKey *rsa.PublicKey
}

// NewEncoder конструктор типа Encoder.
// На вход ожидается путь к файлу публичного ключа в pem-формате.
func NewEncoder(publicKeyFp string) (*Encoder, error) {
	e := &Encoder{}

	// Получить публичный ключ
	content, err := os.ReadFile(publicKeyFp)
	if err != nil {
		return nil, err
	}

	// декодировать pem формат публичного ключа
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("certificate block was not found")
	}

	// парсим публичный ключ
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	e.publicKey = publicKey

	return e, nil
}

// Encode шифрует сообщение посредством RSA OAEP.
func (e *Encoder) Encode(message []byte) ([]byte, error) {
	// шифрование сообщения
	encryptedMsg, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, e.publicKey, message, nil)
	if err != nil {
		return nil, err
	}

	return encryptedMsg, nil
}
