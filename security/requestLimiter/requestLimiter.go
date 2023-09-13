package requestLimiter

import (
	"github.com/didip/tollbooth"
	"github.com/didip/tollbooth/limiter"
	"github.com/go-tron/iris/baseContext"
	"github.com/kataras/iris/v12"
)

func New(lmt *limiter.Limiter) iris.Handler {
	return baseContext.Handler(LimitHandler(lmt))
}

func LimitHandler(lmt *limiter.Limiter) func(ctx *baseContext.Context) {
	return func(ctx *baseContext.Context) {
		if err := tollbooth.LimitByRequest(lmt, ctx.ResponseWriter(), ctx.Request()); err != nil {
			ctx.Error(err)
			return
		}
		ctx.Next()
	}
}
