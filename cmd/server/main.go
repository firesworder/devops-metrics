package main

import (
	"github.com/firesworder/devopsmetrics/internal/server"
	"log"
	"net/http"
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
