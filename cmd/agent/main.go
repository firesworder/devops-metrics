package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/firesworder/devopsmetrics/internal/agent"
)

var (
	buildVersion = "N/A"
	buildDate    = "N/A"
	buildCommit  = "N/A"
)

func main() {
	fmt.Printf("Build version: %s\nBuild date: %s\nBuild commit: %s\n", buildVersion, buildDate, buildCommit)

	agent.ParseEnvArgs()
	agent.InitServerURLByEnv()
	agent.WPool.Start()

	// обработка сигналов системы
	sigClose := make(chan os.Signal, 1)
	signal.Notify(sigClose, syscall.SIGTERM, syscall.SIGINT, syscall.SIGQUIT)
	ctx, cancel := context.WithCancel(context.Background())

	// подготовка тикеров на обновление и отправку
	pollTicker := time.NewTicker(agent.Env.PollInterval)
	reportTicker := time.NewTicker(agent.Env.ReportInterval)
	for {
		select {
		case <-sigClose:
			log.Printf("received signal %v, stopping", sigClose)
			cancel()
			agent.StopAgent()
			log.Println("agent was shutdown gracefully")
			return
		case <-pollTicker.C:
			go agent.UpdateMetrics()
		case <-reportTicker.C:
			go agent.WPool.CreateSendMetricsJob(ctx)
		}
	}
}
