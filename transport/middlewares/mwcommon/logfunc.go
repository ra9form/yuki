package mwcommon

import (
	"context"

	"github.com/ra9form/yuki/server/middlewares/mwcommon"
)

func GetLogFunc(logger interface{}) func(context.Context, string) {
	return mwcommon.GetLogFunc(logger)
}
