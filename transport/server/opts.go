package server

import (
	"github.com/go-chi/chi"
	"google.golang.org/grpc"

	"github.com/ra9form/yuki/server"
	"github.com/ra9form/yuki/server/middleware/http"
	"github.com/ra9form/yuki/transport"
)

// Option is an optional setting applied to the Server.
type Option = server.Option

// WithGRPCOpts sets gRPC server options.
func WithGRPCOpts(opts []grpc.ServerOption) Option {
	return server.WithGRPCOpts(opts)
}

// WithHTTPPort sets HTTP RPC port to listen on.
// Set same port as main to use single port.
func WithHTTPPort(port int) Option {
	return server.WithHTTPPort(port)
}

// WithHTTPMiddlewares sets up HTTP middleware to work with.
func WithHTTPMiddlewares(mws ...http.Middleware) Option {
	return server.WithHTTPMiddlewares(mws...)
}

// WithGRPCUnaryMiddlewares sets up unary middleware for gRPC server.
func WithGRPCUnaryMiddlewares(mws ...grpc.UnaryServerInterceptor) Option {
	return server.WithGRPCUnaryMiddlewares(mws...)
}

// WithGRPCStreamMiddlewares sets up stream middleware for gRPC server.
func WithGRPCStreamMiddlewares(mws ...grpc.StreamServerInterceptor) Option {
	return server.WithGRPCStreamMiddlewares(mws...)
}

// WithHTTPMux sets existing HTTP muxer to use instead of creating new one.
func WithHTTPMux(mux *chi.Mux) Option {
	return server.WithHTTPMux(mux)
}

func WithHTTPRouterMux(mux transport.Router) Option {
	return server.WithHTTPRouterMux(mux)
}
