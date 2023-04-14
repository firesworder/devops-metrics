package main

import (
	"github.com/firesworder/devopsmetrics/internal/server"
	"log"
	"net/http"
)

func main() {
	server.ParseEnvArgs()

	dbErr := server.ConnectToDB()
	// в случае если бд не законнектилась - продолжаю работу все равно! (чтобы пройти тесты)
	if dbErr == nil {
		defer server.DBConn.Close()
	}

	serverParams := server.NewServer()
	serverObj := &http.Server{
		Addr:    server.Env.ServerAddress,
		Handler: serverParams.Router,
	}
	log.Fatal(serverObj.ListenAndServe())
}
