// Package grpcx hosts shared gRPC interceptors used by the plugin runtime.
package grpcx

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/linuxsuren/atest-ext-ai/pkg/logging"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

const requestIDHeader = "x-request-id"

// RequestIDInterceptor injects or propagates a request ID for every gRPC call.
func RequestIDInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		var (
			requestID string
			md        metadata.MD
			ok        bool
		)

		if md, ok = metadata.FromIncomingContext(ctx); !ok {
			md = metadata.New(nil)
		} else {
			md = md.Copy()
		}

		if values := md.Get(requestIDHeader); len(values) > 0 && values[0] != "" {
			requestID = values[0]
		} else {
			requestID = uuid.NewString()
			md.Set(requestIDHeader, requestID)
		}

		ctx = metadata.NewIncomingContext(ctx, md)
		ctx = logging.WithRequestID(ctx, requestID)

		_ = grpc.SetHeader(ctx, metadata.Pairs(requestIDHeader, requestID))
		_ = grpc.SetTrailer(ctx, metadata.Pairs(requestIDHeader, requestID))

		return handler(ctx, req)
	}
}

// LoggingInterceptor emits structured logs for every gRPC call.
func LoggingInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		logger := logging.FromContext(ctx)
		start := time.Now()

		logFields := []any{"method", info.FullMethod}
		if peerInfo, ok := peer.FromContext(ctx); ok && peerInfo.Addr != nil {
			logFields = append(logFields, "peer", peerInfo.Addr.String())
		}

		logger.Info("gRPC request received", logFields...)

		resp, err := handler(ctx, req)
		duration := time.Since(start)
		logFields = append(logFields, "duration_ms", duration.Milliseconds())

		if err != nil {
			logger.Error("gRPC request failed", append(logFields, "error", err)...)
			return resp, err
		}

		logger.Info("gRPC request completed", logFields...)
		return resp, nil
	}
}
