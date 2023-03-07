package main

import (
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/server"
	"net/http"
)

func main() {
	serverObj := &http.Server{
		Addr:    "localhost:8080",
		Handler: http.HandlerFunc(server.CustomHandler),
	}
	err := serverObj.ListenAndServe()
	if err != nil {
		fmt.Println("Произошла ошибка при запуске сервера:", err)
		return
	}
}
