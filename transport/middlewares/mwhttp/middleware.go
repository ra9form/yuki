package mwhttp

import (
	"github.com/ra9form/yuki/server/middlewares/mwhttp"
)

// Middleware is the HTTP middleware type.
// It processes the request (potentially mutating it) and
// gives control to the underlying handler.
type Middleware = mwhttp.Middleware
