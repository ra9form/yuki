package transport

import (
	"google.golang.org/grpc"

	"github.com/ra9form/yuki/transport/httptransport"
	"github.com/ra9form/yuki/transport/swagger"
)

// DescOption modifies the ServiceDesc's behaviour.
type DescOption interface {
	Apply(*httptransport.DescOptions)
}

// WithUnaryInterceptor sets up the interceptor for incoming calls.
func WithUnaryInterceptor(i grpc.UnaryServerInterceptor) DescOption {
	return httptransport.OptionUnaryInterceptor{Interceptor: i}
}

// WithSwaggerOptions sets up default Swagger options for the SwaggerDef().
func WithSwaggerOptions(o ...swagger.Option) DescOption {
	return httptransport.OptionSwaggerOpts{Options: o}
}
