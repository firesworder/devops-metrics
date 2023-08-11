package server

import (
	"context"
	"github.com/firesworder/devopsmetrics/internal"
	"github.com/firesworder/devopsmetrics/internal/message"
	"github.com/firesworder/devopsmetrics/internal/storage"
	pb "github.com/firesworder/devopsmetrics/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"net"
	"testing"
)

func startTestServer(t *testing.T, ts *TempServer) (pb.MetricsClient, func()) {
	serverStarted := make(chan struct{})
	// определяем порт для сервера
	listen, err := net.Listen("tcp", "127.0.0.1:3030")
	require.NoError(t, err)

	// инциал. сервис
	service := &GRPCServer{server: ts}
	// создаем пустой grpc сервер, без опций
	server := grpc.NewServer(grpc.UnaryInterceptor(service.serverInterceptor))
	// регистрируем сервис на сервере
	pb.RegisterMetricsServer(server, service)

	// запуск сервера в горутине
	go func() {
		serverStarted <- struct{}{}
		// запускаем grpc сервер на выделенном порту 'listen'
		if err := server.Serve(listen); err != nil {
			require.NoError(t, err)
		}
	}()
	<-serverStarted

	// создание соединения к запущенному серверу
	conn, err := grpc.Dial(listen.Addr().String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	client := pb.NewMetricsClient(conn)

	closer := func() {
		var err error
		// закрываю grpc сервер
		server.GracefulStop()

		err = conn.Close()
		require.NoError(t, err)
	}
	return client, closer
}

func Test_getMessageMetric(t *testing.T) {
	gaugeV := float64(12.13)
	counterV := int64(15)
	hash := "Ayayaka"

	tests := []struct {
		name          string
		metric        *pb.Metric
		wantMetricMsg *message.Metrics
	}{
		{
			name: "Test 1. Gauge metric",
			metric: &pb.Metric{
				Id:    "RandomValue",
				MType: internal.GaugeTypeName,
				Delta: 0,
				Value: gaugeV,
				Hash:  "d7080abdf2783ee3dfcb9166b59c0693a9795ced688ed7bac2094e694d03be26",
			},
			wantMetricMsg: &message.Metrics{
				ID:    "RandomValue",
				MType: internal.GaugeTypeName,
				Delta: nil,
				Value: &gaugeV,
				Hash:  "d7080abdf2783ee3dfcb9166b59c0693a9795ced688ed7bac2094e694d03be26",
			},
		},
		{
			name: "Test 2. Counter metric",
			metric: &pb.Metric{
				Id:    "PollCount",
				MType: internal.CounterTypeName,
				Delta: counterV,
				Value: 0,
				Hash:  "bc9ffe6bf6dfc772d921dc4889b9db1f07f069c0b018b5e52e3249b9d39f41e2",
			},
			wantMetricMsg: &message.Metrics{
				ID:    "PollCount",
				MType: internal.CounterTypeName,
				Delta: &counterV,
				Value: nil,
				Hash:  "bc9ffe6bf6dfc772d921dc4889b9db1f07f069c0b018b5e52e3249b9d39f41e2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := getMessageMetric(tt.metric)
			require.NoError(t, msg.InitHash(hash))
			assert.Equal(t, tt.wantMetricMsg, msg)
		})
	}
}

func Test_getPbMetric(t *testing.T) {
	gaugeV := float64(12.13)
	counterV := int64(15)

	tests := []struct {
		name       string
		metricMsg  *message.Metrics
		wantMetric *pb.Metric
	}{
		{
			name: "Test 1. Gauge metric",
			metricMsg: &message.Metrics{
				ID:    "RandomValue",
				MType: internal.GaugeTypeName,
				Delta: nil,
				Value: &gaugeV,
				Hash:  "d7080abdf2783ee3dfcb9166b59c0693a9795ced688ed7bac2094e694d03be26",
			},
			wantMetric: &pb.Metric{
				Id:    "RandomValue",
				MType: internal.GaugeTypeName,
				Delta: 0,
				Value: gaugeV,
				Hash:  "d7080abdf2783ee3dfcb9166b59c0693a9795ced688ed7bac2094e694d03be26",
			},
		},
		{
			name: "Test 2. Counter metric",
			metricMsg: &message.Metrics{
				ID:    "PollCount",
				MType: internal.CounterTypeName,
				Delta: &counterV,
				Value: nil,
				Hash:  "bc9ffe6bf6dfc772d921dc4889b9db1f07f069c0b018b5e52e3249b9d39f41e2",
			},
			wantMetric: &pb.Metric{
				Id:    "PollCount",
				MType: internal.CounterTypeName,
				Delta: counterV,
				Value: 0,
				Hash:  "bc9ffe6bf6dfc772d921dc4889b9db1f07f069c0b018b5e52e3249b9d39f41e2",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metric := getPbMetric(*tt.metricMsg)
			assert.Equal(t, tt.wantMetric, metric)
		})
	}
}

func TestGRPCServer_serverInterceptor(t *testing.T) {
	// сбросить изменения в Env
	defer ParseEnvArgs()

	ctx := context.Background()
	type statusInfo struct {
		code    codes.Code
		message string
	}

	tests := []struct {
		name             string
		envTrustedSubnet string
		md               metadata.MD
		wantStatus       statusInfo
	}{
		{
			name:             "Test 1. Trusted subnet is not set.",
			envTrustedSubnet: "",
			md:               metadata.New(map[string]string{}),
			wantStatus:       statusInfo{code: codes.NotFound, message: "metric with name '' not found"},
		},
		{
			name:             "Test 2. X-Real-IP is not set.",
			envTrustedSubnet: "192.168.1.0/24",
			md:               metadata.New(map[string]string{}),
			wantStatus:       statusInfo{code: codes.InvalidArgument, message: "metadata param x-real-ip is not set"},
		},
		{
			name:             "Test 3. X-Real-IP is not in TrustedSubnet.",
			envTrustedSubnet: "192.168.1.0/24",
			md:               metadata.New(map[string]string{"X-Real-IP": "192.122.1.1"}),
			wantStatus:       statusInfo{code: codes.PermissionDenied, message: "user ip is not in trusted subnet"},
		},
		{
			name:             "Test 4. X-Real-IP is in TrustedSubnet.",
			envTrustedSubnet: "192.168.1.0/24",
			md:               metadata.New(map[string]string{"X-Real-IP": "192.168.1.1"}),
			wantStatus:       statusInfo{code: codes.NotFound, message: "metric with name '' not found"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Env.TrustedSubnet = tt.envTrustedSubnet
			tempServer, err := NewTempServer()
			require.NoError(t, err)

			client, closer := startTestServer(t, tempServer)
			defer closer()

			ctxReq := metadata.NewOutgoingContext(ctx, tt.md)
			_, err = client.GetMetric(ctxReq, &pb.GetMetricRequest{})

			errorStatus, _ := status.FromError(err)
			assert.Equal(t, tt.wantStatus.code, errorStatus.Code())
			assert.Equal(t, tt.wantStatus.message, errorStatus.Message())
		})
	}
	Env.TrustedSubnet = ""
}

func TestGRPCServer_GetMetric(t *testing.T) {
	ctx := context.Background()

	tempServer, err := NewTempServer()
	tempServer.MetricStorage = &storage.MemStorage{Metrics: map[string]storage.Metric{
		metric1.Name: *metric1,
		metric2.Name: *metric2,
	}}
	require.NoError(t, err)

	service := &GRPCServer{server: tempServer}

	tests := []struct {
		name       string
		req        *pb.GetMetricRequest
		wantResp   *pb.GetMetricResponse
		wantStatus *status.Status
	}{
		{
			name:       "Test 1. Metric is not present",
			req:        &pb.GetMetricRequest{ID: "Demo"},
			wantResp:   nil,
			wantStatus: status.New(codes.NotFound, "metric with name 'Demo' not found"),
		},
		{
			name:       "Test 2. Gauge metric is present",
			req:        &pb.GetMetricRequest{ID: "PollCount"},
			wantResp:   &pb.GetMetricResponse{Metric: getPbMetric(metric1.GetMessageMetric()), Error: ""},
			wantStatus: nil,
		},
		{
			name:       "Test 3. Counter metric is present",
			req:        &pb.GetMetricRequest{ID: "RandomValue"},
			wantResp:   &pb.GetMetricResponse{Metric: getPbMetric(metric2.GetMessageMetric()), Error: ""},
			wantStatus: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := service.GetMetric(ctx, tt.req)
			s, ok := status.FromError(err)
			require.Equal(t, true, ok)
			assert.Equal(t, tt.wantResp, resp)
			assert.Equal(t, tt.wantStatus, s)
		})
	}
}

func TestGRPCServer_UpdateMetric(t *testing.T) {
	ctx := context.Background()

	tempServer, err := NewTempServer()
	require.NoError(t, err)

	pbMetric1, pbMetric2 := getPbMetric(metric1.GetMessageMetric()), getPbMetric(metric2.GetMessageMetric())
	counter20, gauge235 := int64(20), float64(23.5)

	service := &GRPCServer{server: tempServer}

	tests := []struct {
		name       string
		req        *pb.UpdateMetricRequest
		memStorage storage.MetricRepository
		wantResp   *pb.UpdateMetricResponse
		wantStatus *status.Status
	}{
		{
			name:       "Test 1. Counter metric is not present",
			req:        &pb.UpdateMetricRequest{Metric: pbMetric1},
			memStorage: &storage.MemStorage{Metrics: map[string]storage.Metric{}},
			wantResp:   &pb.UpdateMetricResponse{Metric: pbMetric1},
			wantStatus: nil,
		},
		{
			name:       "Test 2. Gauge metric is not present",
			req:        &pb.UpdateMetricRequest{Metric: pbMetric2},
			memStorage: &storage.MemStorage{Metrics: map[string]storage.Metric{}},
			wantResp:   &pb.UpdateMetricResponse{Metric: pbMetric2},
			wantStatus: nil,
		},
		{
			name: "Test 3. Counter metric is present",
			req: &pb.UpdateMetricRequest{Metric: getPbMetric(message.Metrics{
				ID:    "PollCount",
				MType: internal.CounterTypeName,
				Delta: &counter20,
				Value: nil,
				Hash:  "",
			})},
			memStorage: &storage.MemStorage{
				Metrics: map[string]storage.Metric{
					metric1.Name: *metric1, metric2.Name: *metric2,
				},
			},
			wantResp:   &pb.UpdateMetricResponse{Metric: getPbMetric(metric1upd20.GetMessageMetric())},
			wantStatus: nil,
		},
		{
			name: "Test 4. Gauge metric is present",
			req: &pb.UpdateMetricRequest{Metric: getPbMetric(message.Metrics{
				ID:    "RandomValue",
				MType: internal.GaugeTypeName,
				Delta: nil,
				Value: &gauge235,
				Hash:  "",
			})},
			memStorage: &storage.MemStorage{
				Metrics: map[string]storage.Metric{metric1.Name: *metric1, metric2.Name: *metric2},
			},
			wantResp:   &pb.UpdateMetricResponse{Metric: getPbMetric(metric2upd235.GetMessageMetric())},
			wantStatus: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempServer.MetricStorage = tt.memStorage

			resp, err := service.UpdateMetric(ctx, tt.req)
			s, ok := status.FromError(err)
			require.Equal(t, true, ok)
			assert.Equal(t, tt.wantResp, resp)
			assert.Equal(t, tt.wantStatus, s)
		})
	}
}

func TestGRPCServer_BatchMetricUpdate(t *testing.T) {
	ctx := context.Background()

	tempServer, err := NewTempServer()
	require.NoError(t, err)

	metric1change, err := storage.NewMetric("PollCount", internal.CounterTypeName, int64(20))
	require.NoError(t, err)

	service := &GRPCServer{server: tempServer}
	tests := []struct {
		name           string
		req            *pb.BatchMetricUpdateRequest
		memStorage     storage.MetricRepository
		wantMemStorage storage.MetricRepository
		wantResp       *pb.BatchMetricUpdateResponse
		wantStatus     *status.Status
	}{
		{
			name: "Test 1. Empty metric repository",
			req: &pb.BatchMetricUpdateRequest{Metrics: []*pb.Metric{
				getPbMetric(metric1.GetMessageMetric()),
				getPbMetric(metric2.GetMessageMetric()),
			}},
			memStorage: &storage.MemStorage{Metrics: map[string]storage.Metric{}},
			wantMemStorage: &storage.MemStorage{Metrics: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			}},
			wantResp:   &pb.BatchMetricUpdateResponse{},
			wantStatus: nil,
		},
		{
			name: "Test 2. Filled metric repository",
			req: &pb.BatchMetricUpdateRequest{Metrics: []*pb.Metric{
				getPbMetric(metric1change.GetMessageMetric()),
				getPbMetric(metric2upd235.GetMessageMetric()),
			}},
			memStorage: &storage.MemStorage{Metrics: map[string]storage.Metric{
				metric1.Name: *metric1,
				metric2.Name: *metric2,
			}},
			wantMemStorage: &storage.MemStorage{Metrics: map[string]storage.Metric{
				metric1.Name: *metric1upd20,
				metric2.Name: *metric2upd235,
			}},
			wantResp:   &pb.BatchMetricUpdateResponse{},
			wantStatus: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempServer.MetricStorage = tt.memStorage

			resp, err := service.BatchMetricUpdate(ctx, tt.req)
			s, ok := status.FromError(err)
			require.Equal(t, true, ok)
			assert.Equal(t, tt.wantResp, resp)
			assert.Equal(t, tt.wantStatus, s)
			assert.Equal(t, tt.wantMemStorage, tempServer.MetricStorage)
		})
	}
}
