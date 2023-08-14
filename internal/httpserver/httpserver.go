package httpserver

import (
	"github.com/firesworder/devopsmetrics/internal/server"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
)

// HTTPServer реализует серверную логику.
// Всё взаимодействие с серверной частью происходит через него.
type HTTPServer struct {
	// todo: нужен здесь server.Env или нет?
	server *server.TempServer
	Router chi.Router
}

func NewHTTPServer(server *server.TempServer) *HTTPServer {
	s := HTTPServer{server: server}
	s.Router = s.newRouter()
	return &s
}

// newRouter определяет и возвращает роутер для сервера.
func (hs *HTTPServer) newRouter() chi.Router {
	r := chi.NewRouter()

	// todo: r.Use(hs.checkRequestSubnet)
	r.Use(hs.gzipDecompressor)
	r.Use(hs.gzipCompressor)
	r.Use(hs.decryptMessage)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Route("/", func(r chi.Router) {
		r.Get("/", hs.handlerShowAllMetrics)
		r.Get("/value/{typeName}/{metricName}", hs.handlerGet)
		r.Get("/ping", hs.handlerPing)
		r.Post("/updates/", hs.handlerBatchUpdate)
		r.Post("/update/{typeName}/{metricName}/{metricValue}", hs.handlerAddUpdateMetric)
		r.Post("/update/", hs.handlerJSONAddUpdateMetric)
		r.Post("/value/", hs.handlerJSONGetMetric)
	})
	return r
}
