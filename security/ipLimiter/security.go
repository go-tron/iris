package ipLimiter

import (
	"github.com/go-tron/iris/baseContext"
	"github.com/go-tron/rate-limiter"
	"github.com/kataras/iris/v12"
)

func New(fl *rateLimiter.RateLimiter) iris.Handler {
	return baseContext.Handler(func(ctx *baseContext.Context) {
		if _, err := fl.Check(ctx.GetIP()); err != nil {
			if ctx.GetHeader("referer") != "" {
				ctx.Error(err)
			} else {
				ctx.ErrorView(err)
			}
			return
		}
		ctx.Next()
	})
}
