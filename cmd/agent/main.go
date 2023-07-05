package main

import (
	"context"
	"fmt"
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
	defer agent.WPool.Close()
	// подготовка тикеров на обновление и отправку
	pollTicker := time.NewTicker(agent.Env.PollInterval)
	reportTicker := time.NewTicker(agent.Env.ReportInterval)
	for {
		select {
		case <-pollTicker.C:
			go agent.UpdateMetrics()
		case <-reportTicker.C:
			go agent.WPool.CreateSendMetricsJob(context.Background())
		}
	}
}
