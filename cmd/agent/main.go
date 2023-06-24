package main

import (
	"context"
	"time"

	"github.com/firesworder/devopsmetrics/internal/agent"
)

func main() {
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
