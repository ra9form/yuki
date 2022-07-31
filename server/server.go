package server

import (
	"bytes"
	"io"
	"net/http"

	"github.com/pkg/errors"

	"github.com/ra9form/yuki/transport"
)

const maxRunErrs = 5

// Server is a transport server.
type Server struct {
	opts      *serverOpts
	listeners *listenerSet
	srv       *serverSet
}

// NewServer creates a Server listening on the rpcPort.
// Pass additional Options to mutate its behaviour.
// By default, HTTP JSON handler and gRPC are listening on the same
// port, admin port is p+2 and profile port is p+4.
func NewServer(rpcPort int, opts ...Option) *Server {
	serverOpts := defaultServerOpts(rpcPort)
	for _, opt := range opts {
		opt(serverOpts)
	}

	return &Server{
		opts: serverOpts,
	}
}

// Run starts processing requests to the service.
// It blocks indefinitely, run asynchronously to do anything after that.
func (srv *Server) Run(svc transport.Service) error {
	desc := svc.GetDescription()

	var err error

	srv.listeners, err = newListenerSet(srv.opts)
	if err != nil {
		return errors.Wrap(err, "couldn't create listeners")
	}

	srv.srv = newServerSet(srv.listeners, srv.opts)
	// Inject static Swagger as root handler
	srv.srv.http.HandleFunc("/swagger.json", func(w http.ResponseWriter, req *http.Request) {
		io.Copy(w, bytes.NewReader(desc.SwaggerDef()))
	})

	// apply gRPC interceptor
	if d, ok := desc.(transport.ConfigurableServiceDesc); ok {
		d.Apply(transport.WithUnaryInterceptor(srv.opts.GRPCUnaryInterceptor))
	}

	// Register everything
	desc.RegisterHTTP(srv.srv.http)
	desc.RegisterGRPC(srv.srv.grpc)

	return srv.run()
}

func (srv *Server) run() error {
	errChan := make(chan error, maxRunErrs)

	if srv.listeners.mainListener != nil {
		go func() {
			err := srv.listeners.mainListener.Serve()
			errChan <- err
		}()
	}

	go func() {
		err := http.Serve(srv.listeners.HTTP, srv.srv.http)
		errChan <- err
	}()

	go func() {
		err := srv.srv.grpc.Serve(srv.listeners.GRPC)
		errChan <- err
	}()

	return <-errChan
}

// Stop stops the server gracefully.
func (srv *Server) Stop() {
	// TODO grace HTTP
	srv.srv.grpc.GracefulStop()
}
