package main

import (
	"fmt"
	"github.com/firesworder/devopsmetrics/internal/server"
	"net/http"
)

func main() {
	metricHandler := server.NewDefaultMetricHandler()
	serverObj := &http.Server{
		Addr:    "localhost:8080",
		Handler: metricHandler,
	}
	err := serverObj.ListenAndServe()
	if err != nil {
		fmt.Println("Произошла ошибка при запуске сервера:", err)
		return
	}
}
