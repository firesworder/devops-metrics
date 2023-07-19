package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/firesworder/devopsmetrics/internal/server"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)
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
