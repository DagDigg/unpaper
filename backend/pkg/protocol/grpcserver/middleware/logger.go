package middleware

import (
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// codeToLevel redirects OK to DEBUG level logging instead of INFO
func codeToLevel(code codes.Code) zapcore.Level {
	if code == codes.OK {
		// It's DEBUG
		return zap.DebugLevel
	}
	return grpc_zap.DefaultCodeToLevel(code)
}

// Shared options for the logger, with a custom gRPC code to log level function.
func getBaseZapOptions() []grpc_zap.Option {
	return []grpc_zap.Option{
		grpc_zap.WithLevels(codeToLevel),
	}
}

// AddStreamLogging returns a StreamServerInterceptor for zap logging
func AddStreamLogging(logger *zap.Logger) grpc.StreamServerInterceptor {
	o := getBaseZapOptions()
	return grpc_middleware.ChainStreamServer(
		grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		grpc_zap.StreamServerInterceptor(logger, o...),
	)
}

// AddUnaryLogging returns a UnaryServerInterceptor for zap logging
func AddUnaryLogging(logger *zap.Logger) grpc.UnaryServerInterceptor {
	o := getBaseZapOptions()
	return grpc_middleware.ChainUnaryServer(
		grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		grpc_zap.UnaryServerInterceptor(logger, o...),
	)
}
