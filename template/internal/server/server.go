package server

import (
	"context"

	kratosHttp "github.com/go-kratos/kratos/v2/transport/http"
	"go.uber.org/fx"
)

// AppServer wraps the HTTP server for lifecycle management
type AppServer struct {
	httpServer *kratosHttp.Server
	grpcServer *GRPCServer
}

// NewAppServer creates a new application server
func NewAppServer(httpServer *kratosHttp.Server, grpcServer *GRPCServer) *AppServer {
	return &AppServer{
		httpServer: httpServer,
		grpcServer: grpcServer,
	}
}

// Start starts the application server
func (s *AppServer) Start() error {
	if s.grpcServer != nil {
		go func() {
			// If gRPC exits, it's fatal for the process. This matches HTTP's behavior
			// where a start error will bubble up via panic in main.
			if err := s.grpcServer.Start(context.Background()); err != nil {
				panic(err)
			}
		}()
	}
	return s.httpServer.Start(context.Background())
}

// Stop stops the application server
func (s *AppServer) Stop(ctx context.Context) error {
	if s.grpcServer != nil {
		_ = s.grpcServer.Stop(ctx)
	}
	return s.httpServer.Stop(ctx)
}

// Module provides server dependencies
var Module = fx.Options(
	fx.Provide(
		NewHTTPServer,
		NewGRPCServer,
		NewAppServer,
	),
)
