package server

import (
	"context"
	"fmt"
	"net"

	"spark/internal/conf"
	"spark/pkg/clog"
	"spark/pkg/metrics"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

// GRPCServer wraps a gRPC server and its listener for lifecycle management.
type GRPCServer struct {
	addr   string
	lis    net.Listener
	server *grpc.Server
	logger clog.Logger
}

func NewGRPCServer(c *conf.Bootstrap, logger clog.Logger) (*GRPCServer, error) {
	addr := ":9989"
	if c != nil && c.Server != nil && c.Server.Grpc != nil && c.Server.Grpc.Addr != "" {
		addr = c.Server.Grpc.Addr
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("listen grpc %s: %w", addr, err)
	}

	s := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()),
		grpc.ChainUnaryInterceptor(
			metrics.UnaryServerInterceptor(),
		),
	)

	return &GRPCServer{
		addr:   addr,
		lis:    lis,
		server: s,
		logger: logger,
	}, nil
}

func (s *GRPCServer) Start(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	s.logger.Info(ctx, "Starting gRPC server...", clog.String("addr", s.addr))
	return s.server.Serve(s.lis)
}

func (s *GRPCServer) Stop(ctx context.Context) error {
	if s == nil || s.server == nil {
		return nil
	}
	s.logger.Info(ctx, "Stopping gRPC server...")
	s.server.GracefulStop()
	if s.lis != nil {
		_ = s.lis.Close()
	}
	return nil
}
