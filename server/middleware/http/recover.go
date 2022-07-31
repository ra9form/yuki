package http

import (
	"context"
	"net/http"
	"runtime/debug"

	"github.com/pkg/errors"

	"github.com/ra9form/yuki/transport/httpruntime"
)

type ctxErrorLogger interface {
	Errorf(ctx context.Context, format string, fields ...any)
}

// Recover recovers HTTP server from handlers' panics.
func Recover(logger ctxErrorLogger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					stack := string(debug.Stack())
					httpruntime.SetError(
						r.Context(),
						r, w,
						errors.Errorf("recovered from panic: %v\nstack: %v", rec, stack),
					)
					logger.Errorf(r.Context(), "recovered from panic: %v\nstack: %v", r, stack)
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
