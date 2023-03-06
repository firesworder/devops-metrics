package main

import (
	"github.com/firesworder/devopsmetrics/internal/agent"
	"time"
)

const pollInterval = 2 * time.Second
const reportInterval = 10 * time.Second

func main() {
	// подготовка тикеров на обновление и отправку
	pollTicker := time.NewTicker(pollInterval)
	reportTicker := time.NewTicker(reportInterval)
	for {
		select {
		case <-pollTicker.C:
			agent.UpdateMetrics()
		case <-reportTicker.C:
			agent.SendMetrics()
		}
	}
}
