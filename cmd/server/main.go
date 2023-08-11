package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

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

	// обработка сигналов системы
	sigClose := make(chan os.Signal, 1)
	signal.Notify(sigClose, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	serverParams, err := server.NewTempServer()
	if err != nil {
		log.Fatal(err)
	}
	httpServer := server.NewHTTPServer(serverParams)
	serverObj := &http.Server{
		Addr:    server.Env.ServerAddress,
		Handler: httpServer.Router,
	}
	go func() {
		<-sigClose
		if err := serverObj.Shutdown(context.Background()); err != nil {
			// ошибки закрытия Listener
			log.Printf("HTTP server Shutdown: %v", err)
		}
		serverStopCtx()
	}()
	err = serverObj.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
	<-serverCtx.Done()
	log.Println("server was shutdown gracefully")
}
