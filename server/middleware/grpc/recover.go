package grpc

import (
	"runtime/debug"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ctxErrorLogger interface {
	Errorf(ctx context.Context, format string, fields ...any)
}

// UnaryPanicHandler handles panics for UnaryHandlers.
func UnaryPanicHandler(logger ctxErrorLogger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				err = status.Errorf(codes.Internal, "panic: %v", r)
				logger.Errorf(ctx, "recovered from panic: %v\nstack: %v", r, string(debug.Stack()))
			}
		}()

		return handler(ctx, req)
	}
}

// StreamPanicHandler handles panics for StreamHandlers.
func StreamPanicHandler(logger ctxErrorLogger) grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		stream grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) (err error) {
		defer func() {
			if r := recover(); r != nil {
				err = status.Errorf(codes.Internal, "panic: %v", r)
				logger.Errorf(stream.Context(), "recovered from panic: %v\nstack: %v", r, string(debug.Stack()))
			}
		}()

		return handler(srv, stream)
	}
}
