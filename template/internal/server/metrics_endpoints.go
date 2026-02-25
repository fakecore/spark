package server

import (
	stdhttp "net/http"

	"spark/pkg/metrics"

	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func registerMetricsEndpoint(srv *kratosHttp.Server) {
	if srv == nil {
		return
	}

	metrics.Init()
	h := promhttp.Handler()

	srv.Handle("/metrics", stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		h.ServeHTTP(w, r)
	}))
}
