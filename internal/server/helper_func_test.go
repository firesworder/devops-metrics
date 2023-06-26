package server

import (
	"context"
	"database/sql"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/storage"
	"io"
	"net/http"
	"strings"
)

var (
	exMetricCounter, _ = storage.NewMetric("PollCount", internal.CounterTypeName, int64(10))
	exMetricGauge1, _  = storage.NewMetric("RandomValue", internal.GaugeTypeName, 12.133)
	exMetricGauge2, _  = storage.NewMetric("Alloc", internal.GaugeTypeName, 7.77)
)

func getMetricsMap() map[string]storage.Metric {
	return map[string]storage.Metric{
		exMetricCounter.Name: *exMetricCounter,
		exMetricGauge1.Name:  *exMetricGauge1,
		exMetricGauge2.Name:  *exMetricGauge2,
	}
}

func resetDBState(dbConn *sql.DB) {
	ctx := context.Background()
	// подготовка состояния таблицы
	dbConn.ExecContext(ctx, "DELETE FROM metrics")
	for _, metric := range getMetricsMap() {
		mN, mV, mT := metric.GetMetricParamsString()
		dbConn.ExecContext(ctx, "INSERT INTO metrics(m_name, m_value, m_type) VALUES($1, $2, $3)", mN, mV, mT)
	}
}

func getServer(useDB bool) *Server {
	if useDB {
		Env.DatabaseDsn = "postgresql://postgres:admin@localhost:5432/devops"
	} else {
		Env.DatabaseDsn = ""
	}

	s, err := NewServer()
	if err != nil {
		panic(err)
	}
	s.LayoutsDir = "./html_layouts/"
	if !useDB {
		s.MetricStorage = storage.NewMemStorage(getMetricsMap())
	}
	return s
}

func sendRequest(method, url, contentType, content string) (int, string, string) {
	// создаю реквест
	req, _ := http.NewRequest(method, url, strings.NewReader(content))
	req.Header.Set("Content-Type", contentType)

	// делаю реквест на дефолтном клиенте
	resp, _ := http.DefaultClient.Do(req)

	// читаю ответ сервера
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)

	return resp.StatusCode, resp.Header.Get("Content-Type"), string(respBody)
}
