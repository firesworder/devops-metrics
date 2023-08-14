package httpserver

import (
	"bytes"
	"compress/gzip"
	"crypto/rsa"
	"errors"
	"io"
	"net/http"
	"strings"
)

// gzipResponseWriter для реализации gzipCompressor.
type gzipResponseWriter struct {
	http.ResponseWriter // нужен, чтобы хандлеры не спотыкались об отсутствие возм.установить header например.
	Writer              io.Writer
}

func (w gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

// gzipDecompressor - middleware для обработки входящих запросов с gzip сжатием.
func (hs *HTTPServer) gzipDecompressor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if strings.Contains(request.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(request.Body)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}
			request.Body = gz
			defer gz.Close()
		}
		next.ServeHTTP(writer, request)
	})
}

// gzipDecompressor - middleware для gzip сжатия исходящих запросов.
func (hs *HTTPServer) gzipCompressor(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		// если не допускает сжатие - ничего не делать
		if !strings.Contains(request.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(writer, request)
			return
		}

		// создаю gzipWriter
		gzipWriter := gzip.NewWriter(writer)
		defer gzipWriter.Close()

		// оборачиваю ответ в gzip
		writer.Header().Set("Content-Encoding", "gzip")
		next.ServeHTTP(gzipResponseWriter{ResponseWriter: writer, Writer: gzipWriter}, request)
	})
}

// decryptMessage - middleware для расшифр. в асимм.шифровании
func (hs *HTTPServer) decryptMessage(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if hs.server.Decoder != nil {
			body, err := io.ReadAll(request.Body)
			if err != nil {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
			}

			r, err := hs.server.Decoder.Decode(body)
			// если есть ошибка и это не ошибка расшифровки - выбросить http ошибку
			// если ошибка расшифровки - ничего не делать(оставить изначальный request.Body)
			if err != nil && !errors.Is(err, rsa.ErrDecryption) {
				http.Error(writer, err.Error(), http.StatusInternalServerError)
				return
				// если ошибок нет - заменить reader на расшифр.сообщение
			} else if err == nil {
				reader := io.NopCloser(bytes.NewReader(r))
				request.Body = reader
			}
		}
		next.ServeHTTP(writer, request)
	})
}

// todo: implement me!
func (hs *HTTPServer) checkRequestSubnet(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		panic("implement me!")
		next.ServeHTTP(writer, request)
	})
}
