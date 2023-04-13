package main

import (
	"github.com/firesworder/devopsmetrics/internal/server"
	"log"
	"net/http"
)

func main() {
	server.ParseEnvArgs()

	dbErr := server.ConnectToDB()
	if dbErr != nil {
		log.Fatal(dbErr)
	}
	defer server.DBConn.Close()

	serverParams := server.NewServer()
	serverObj := &http.Server{
		Addr:    server.Env.ServerAddress,
		Handler: serverParams.Router,
	}
	log.Fatal(serverObj.ListenAndServe())
}
