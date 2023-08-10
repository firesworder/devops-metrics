package server

import (
	"context"
	"errors"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	pb "github.com/firesworder/devopsmetrics/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
)

func getMessageMetric(metric *pb.Metric) *message.Metrics {
	msg := message.Metrics{
		ID:    metric.Id,
		MType: metric.MType,
		Hash:  metric.Hash,
	}

	if metric.MType == internal.GaugeTypeName {
		msg.Value = &metric.Value
	} else if metric.MType == internal.CounterTypeName {
		msg.Delta = &metric.Delta
	}
	return &msg
}

func getPbMetric(msg message.Metrics) *pb.Metric {
	metric := pb.Metric{
		Id:    msg.ID,
		MType: msg.MType,
		Hash:  msg.Hash,
	}

	if metric.MType == internal.GaugeTypeName && msg.Value != nil {
		metric.Value = *msg.Value
	} else if metric.MType == internal.CounterTypeName {
		metric.Delta = *msg.Delta
	}
	return &metric
}

type GRPCServer struct {
	// нужно встраивать тип pb.Unimplemented<TypeName>
	// для совместимости с будущими версиями
	pb.UnimplementedMetricsServer

	server *TempServer
}

func (gs *GRPCServer) serverInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (resp interface{}, err error) {
	if gs.server.TrustedSubnet != nil {
		// распарсить метаданные
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Errorf(codes.Internal, "can not access request metadata")
		}

		// получить значение X-Real-IP
		xRealIP := md.Get("X-Real-IP")
		if len(xRealIP) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "metadata param x-real-ip is not set")
		}

		// спарсить IP из значения
		userIP := net.ParseIP(xRealIP[0])
		if userIP == nil {
			return nil, status.Errorf(codes.InvalidArgument, "metadata param x-real-ip is not valid")
		}

		// применить маску к userIP и сравнить с trustedSubnetIP
		maskedUserIP := userIP.Mask(gs.server.TrustedSubnet.Mask)
		if maskedUserIP.String() != gs.server.TrustedSubnet.IP.String() {
			return nil, status.Errorf(codes.PermissionDenied, "user ip is not in trusted subnet")
		}
	}
	return handler(ctx, req)
}

func (gs *GRPCServer) GetMetric(ctx context.Context, in *pb.GetMetricRequest) (*pb.GetMetricResponse, error) {
	var response pb.GetMetricResponse
	var err error

	metric, err := gs.server.MetricStorage.GetMetric(ctx, in.ID)
	if err != nil {
		if errors.Is(err, storage.ErrMetricNotFound) {
			return nil, status.Errorf(codes.NotFound, "metric with name '%s' not found", in.ID)
		} else {
			return nil, status.Errorf(codes.Internal, err.Error())
		}
	}

	responseMsg := metric.GetMessageMetric()
	if Env.Key != "" {
		err = responseMsg.InitHash(Env.Key)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
	}

	response.Metric = getPbMetric(responseMsg)
	return &response, nil
}
func (gs *GRPCServer) UpdateMetric(ctx context.Context, in *pb.UpdateMetricRequest) (*pb.UpdateMetricResponse, error) {
	var response pb.UpdateMetricResponse
	var metric *storage.Metric
	var err error

	metricMessage := getMessageMetric(in.Metric)
	if Env.Key != "" {
		var isHashCorrect bool
		isHashCorrect, err = metricMessage.CheckHash(Env.Key)
		if err != nil {
			return nil, status.Error(codes.Internal, err.Error())
		} else if !isHashCorrect {
			return nil, status.Error(codes.InvalidArgument, "hash is not correct")
		}
	}

	metric, err = storage.NewMetricFromMessage(metricMessage)
	if err != nil {
		if errors.Is(err, storage.ErrUnhandledValueType) {
			return nil, status.Error(codes.Unimplemented, err.Error())
		} else {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
	}

	err = gs.server.MetricStorage.UpdateOrAddMetric(ctx, *metric)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}
	if err = gs.server.syncSaveMetricStorage(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	*metric, err = gs.server.MetricStorage.GetMetric(ctx, metric.Name)
	if err != nil {
		// ошибка не должна произойти, но мало ли
		return nil, status.Error(codes.Internal, "metric was not updated:"+err.Error())
	}

	responseMsg := metric.GetMessageMetric()
	if Env.Key != "" {
		err = responseMsg.InitHash(Env.Key)
		if err != nil {
			return nil, status.Errorf(codes.NotFound, err.Error())
		}
	}

	response.Metric = getPbMetric(responseMsg)
	return &response, nil
}
func (gs *GRPCServer) BatchMetricUpdate(ctx context.Context, in *pb.BatchMetricUpdateRequest) (*pb.BatchMetricUpdateResponse, error) {
	var response pb.BatchMetricUpdateResponse
	var metrics []storage.Metric
	var err error

	for _, mm := range in.Metrics {
		metricMessage := getMessageMetric(mm)

		if Env.Key != "" {
			var isHashCorrect bool
			isHashCorrect, err = metricMessage.CheckHash(Env.Key)
			if err != nil {
				return nil, status.Error(codes.Internal, err.Error())
			} else if !isHashCorrect {
				return nil, status.Error(codes.InvalidArgument, "hash is not correct")
			}
		}

		var m *storage.Metric
		m, err = storage.NewMetricFromMessage(metricMessage)
		if err != nil {
			if errors.Is(err, storage.ErrUnhandledValueType) {
				return nil, status.Error(codes.Unimplemented, err.Error())
			} else {
				return nil, status.Error(codes.InvalidArgument, err.Error())
			}
		}
		metrics = append(metrics, *m)
	}

	if err = gs.server.MetricStorage.BatchUpdate(ctx, metrics); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	if err = gs.server.syncSaveMetricStorage(); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &response, nil
}
