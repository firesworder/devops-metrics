package main

import (
	"github.com/firesworder/devopsmetrics/internal/agent"
	"time"
)

func main() {
	agent.ParseEnvArgs()
	agent.InitServerURLByEnv()
	// подготовка тикеров на обновление и отправку
	pollTicker := time.NewTicker(agent.Env.PollInterval)
	reportTicker := time.NewTicker(agent.Env.ReportInterval)
	for {
		select {
		case <-pollTicker.C:
			go agent.UpdateMetrics()
		case <-reportTicker.C:
			go agent.SendMetrics()
		}
	}
}
