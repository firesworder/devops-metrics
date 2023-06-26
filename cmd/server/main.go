package main

import (
	"log"
	"net/http"

	"github.com/firesworder/devopsmetrics/internal/server"
)

func main() {
	server.ParseEnvArgs()

	serverParams, err := server.NewServer()
	if err != nil {
		log.Fatal(err)
	}
	serverObj := &http.Server{
		Addr:    server.Env.ServerAddress,
		Handler: serverParams.Router,
	}
	log.Fatal(serverObj.ListenAndServe())
}
