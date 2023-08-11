package agent

import (
	"context"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	pb "github.com/firesworder/devopsmetrics/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
)

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

type GRPCAgent struct {
	conn       *grpc.ClientConn
	grpcClient pb.MetricsClient
}

func Init(serverAddr string) (*GRPCAgent, error) {
	a := GRPCAgent{}
	conn, err := grpc.Dial(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	a.grpcClient = pb.NewMetricsClient(conn)
	return &a, nil
}

func (a *GRPCAgent) sendMetricBatch(metrics map[string]interface{}) {
	var metricsToSend []*pb.Metric
	var msg message.Metrics
	for mN, mV := range metrics {
		msg = message.Metrics{}

		msg.ID = mN
		switch value := mV.(type) {
		case gauge:
			msg.MType = internal.GaugeTypeName
			float64Val := float64(value)
			msg.Value = &float64Val
		case counter:
			msg.MType = internal.CounterTypeName
			int64Val := int64(value)
			msg.Delta = &int64Val
		default:
			log.Printf("unhandled metric type '%T'", value)
			return
		}

		if Env.Key != "" {
			err := msg.InitHash(Env.Key)
			if err != nil {
				log.Println(err)
				return
			}
		}

		metricsToSend = append(metricsToSend, getPbMetric(msg))
	}
	_, err := a.grpcClient.BatchMetricUpdate(context.Background(), &pb.BatchMetricUpdateRequest{Metrics: metricsToSend})
	if err != nil {
		log.Println(err)
	}
}

func (a *GRPCAgent) Close() error {
	return a.conn.Close()
}
