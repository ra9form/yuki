package mwgrpc

import (
	"google.golang.org/grpc"

	"github.com/ra9form/yuki/server/middlewares/mwgrpc"
)

// UnaryPanicHandler handles panics for UnaryHandlers.
func UnaryPanicHandler(logger interface{}) grpc.UnaryServerInterceptor {
	return mwgrpc.UnaryPanicHandler(logger)
}

// StreamPanicHandler handles panics for StreamHandlers.
func StreamPanicHandler(logger interface{}) grpc.StreamServerInterceptor {
	return mwgrpc.StreamPanicHandler(logger)
}
