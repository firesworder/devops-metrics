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

// Decoder RSA декодер, при помощи прив.ключа расшифровывает сообщения.
type Decoder struct {
	privateKey *rsa.PrivateKey
}

// NewDecoder конструктор типа Decoder.
// На вход ожидается путь к файлу приватного ключа в pem-формате.
func NewDecoder(privateKeyFp string) (*Decoder, error) {
	d := &Decoder{}

	// Получить приватного ключ
	content, err := os.ReadFile(privateKeyFp)
	if err != nil {
		return nil, err
	}

	// декодировать pem формат приватного ключа
	block, _ := pem.Decode(content)
	if block == nil {
		return nil, fmt.Errorf("private key is empty")
	}

	// парсим приватный ключ
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	d.privateKey = privateKey

	return d, nil
}

// Decode дешифрует сообщение посредством RSA OAEP.
func (d *Decoder) Decode(encryptedMsg []byte) ([]byte, error) {
	// шифрование сообщения
	msg, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, d.privateKey, encryptedMsg, nil)
	if err != nil {
		return nil, err
	}
	return msg, nil
}
