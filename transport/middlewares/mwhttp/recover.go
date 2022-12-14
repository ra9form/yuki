package mwhttp

import (
	"github.com/ra9form/yuki/server/middlewares/mwhttp"
)

// Recover recovers HTTP server from handlers' panics.
func Recover(logger interface{}) Middleware {
	return mwhttp.Recover(logger)
}
