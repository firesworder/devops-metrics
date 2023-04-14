package main

import (
	"github.com/firesworder/devopsmetrics/internal/server"
	"log"
	"net/http"
)

func main() {
	server.ParseEnvArgs()
	if server.Env.DatabaseDsn != "" {
		log.Fatal(server.ConnectToDB())
	}

	serverParams := server.NewServer()
	serverObj := &http.Server{
		Addr:    server.Env.ServerAddress,
		Handler: serverParams.Router,
	}
	log.Fatal(serverObj.ListenAndServe())
}
