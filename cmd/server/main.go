package main

import (
	"github.com/firesworder/devopsmetrics/internal/server"
	"log"
	"net/http"
)

func main() {
	serverParams := server.NewServer()
	serverObj := &http.Server{
		Addr:    "localhost:8080",
		Handler: serverParams.Router,
	}
	log.Fatal(serverObj.ListenAndServe())
}
