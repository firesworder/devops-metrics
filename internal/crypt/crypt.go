// Package crypt реализует шифрование\дешифрование сообщения методом RSA с использованием публичного\приватного
// ключа соответственно.
package crypt

import (
	"bytes"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"os"
)

// Encode шифрует сообщение посредством RSA OAEP.
// Возвращает зашифрованное сообщение.
func Encode(publicKeyFp string, message []byte) ([]byte, error) {
	// Получить публичный ключ
	content, err := os.ReadFile(publicKeyFp)
	if err != nil {
		return nil, err
	}
	// декодировать pem формат сертификата(!), содержащего публичный ключ
	block, _ := pem.Decode(content)
	if block == nil || block.Type != "RSA PUBLIC KEY" {
		return nil, fmt.Errorf("certificate block was not found")
	}
	// парсим x509 сертификат и получаем из него публичный ключ
	publicKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// шифрование сообщения
	encryptedMsg, err := rsa.EncryptOAEP(sha256.New(), rand.Reader, publicKey, []byte(message), nil)
	if err != nil {
		return nil, err
	}

	return encryptedMsg, nil
}

// Decode дешифрует сообщение посредством RSA OAEP.
// Возвращает дешифрованное сообщение.
func Decode(privateKeyFp string, encryptedMsg []byte) ([]byte, error) {
	// Получить публичный ключ
	content, err := os.ReadFile(privateKeyFp)
	if err != nil {
		return nil, err
	}
	// декодировать pem формат сертификата(!), содержащего публичный ключ
	block, _ := pem.Decode(content)
	if block == nil || block.Type != "RSA PRIVATE KEY" {
		return nil, fmt.Errorf("private key block was not found")
	}
	// парсим приватный ключ
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// шифрование сообщения
	msg, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, privateKey, encryptedMsg, nil)
	if err != nil {
		return nil, err
	}

	return msg, nil
}

type Reader struct {
	decryptedMsg io.Reader
}

func NewReader(privateKeyFp string, innerReader io.ReadCloser) (*Reader, error) {
	r := &Reader{}

	body, err := io.ReadAll(innerReader)
	if err != nil {
		return nil, err
	}
	defer innerReader.Close()

	decryptedMsgBytes, err := Decode(privateKeyFp, body)
	if err != nil {
		return nil, err
	}
	r.decryptedMsg = bytes.NewReader(decryptedMsgBytes)

	return r, nil
}

func (r *Reader) Read(p []byte) (n int, err error) {
	return r.decryptedMsg.Read(p)
}

func (r *Reader) Close() error {
	return nil
}
