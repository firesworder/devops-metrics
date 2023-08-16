package server

import (
	"context"
	"errors"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
)

func (s *Server) GetAllMetrics(ctx context.Context) (map[string]storage.Metric, error) {
	return s.metricStorage.GetAll(ctx)
}

func (s *Server) Ping(ctx context.Context) error {
	if s.dbConn == nil {
		return ErrNoDBConn
	}
	return s.dbConn.Ping()
}

func (s *Server) UpdateMetric(ctx context.Context, metricMessage message.Metrics) (*message.Metrics, error) {
	var metric *storage.Metric
	var err error

	if err = message.CheckHash(metricMessage, s.env.Key); err != nil && !errors.Is(err, message.ErrEmptyKey) {
		return nil, err
	}

	metric, err = storage.NewMetricFromMessage(&metricMessage)
	if err != nil {
		return nil, err
	}

	err = s.metricStorage.UpdateOrAddMetric(ctx, *metric)
	if err != nil {
		return nil, err
	}

	if err = s.syncSaveMetricStorage(); err != nil {
		return nil, err
	}

	*metric, err = s.metricStorage.GetMetric(ctx, metric.Name)
	if err != nil {
		return nil, err
	}

	responseMsg := metric.GetMessageMetric()
	if err = responseMsg.InitHash(s.env.Key); err != nil && !errors.Is(err, message.ErrEmptyKey) {
		return nil, err
	}
	return &responseMsg, nil
}

func (s *Server) GetMetric(ctx context.Context, metricMessage message.Metrics) (*message.Metrics, error) {
	var err error
	metric, err := s.metricStorage.GetMetric(ctx, metricMessage.ID)
	if err != nil {
		return nil, err
	}

	metricMessage = metric.GetMessageMetric()
	if err = metricMessage.InitHash(s.env.Key); err != nil && !errors.Is(err, message.ErrEmptyKey) {
		return nil, err
	}
	return &metricMessage, nil
}

func (s *Server) BatchUpdate(ctx context.Context, metricMessagesBatch []message.Metrics) error {
	var metrics []storage.Metric

	for _, metricMessage := range metricMessagesBatch {
		var err error
		if err = message.CheckHash(metricMessage, s.env.Key); err != nil {
			return err
		}

		m, err := storage.NewMetricFromMessage(&metricMessage)
		if err != nil {
			return err
		}
		metrics = append(metrics, *m)
	}

	if err := s.metricStorage.BatchUpdate(ctx, metrics); err != nil {
		return err
	}

	if err := s.syncSaveMetricStorage(); err != nil {
		return err
	}
	return nil
}
