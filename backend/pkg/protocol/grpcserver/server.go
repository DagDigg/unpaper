package grpcserver

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/DagDigg/unpaper/backend/pkg/logger"
	"github.com/DagDigg/unpaper/backend/pkg/protocol/grpcserver/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
)

// Server embeds a grpc.Server and its context
type Server struct {
	Srv *grpc.Server
	Ctx context.Context
}

// Get creates a new grpc server, adding middleware options, and returns it
func Get(ctx context.Context, creds credentials.TransportCredentials) *Server {
	// Make sure that log statements internal to gRPC library are logged using the zapLogger as well.
	grpc_zap.ReplaceGrpcLoggerV2(logger.Log)

	// register service
	server := grpc.NewServer(
		grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(
			middleware.AddStreamLogging(logger.Log),
		)),
		grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(
			middleware.AddUnaryLogging(logger.Log),
		)),
		// grpc.Creds(creds),
	)

	return &Server{
		Srv: server,
		Ctx: ctx,
	}
}

// ListenForShutdown creates a goroutine that listens
// for OS shutdown signals and stops the server
func (server *Server) ListenForShutdown() {
	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		for range c {
			// sig is a ^C, handle it
			logger.Log.Warn("shutting down gRPC server . . .")
			server.Srv.GracefulStop()
			<-server.Ctx.Done()
		}
	}()
}
