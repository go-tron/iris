package recover

import (
	"errors"
	"fmt"
	baseError "github.com/go-tron/base-error"
	"github.com/go-tron/config"
	"github.com/go-tron/iris/baseContext"
	"github.com/kataras/iris/v12"
	"reflect"
)

func New() iris.Handler {
	return baseContext.Handler(Recover())
}

func Recover() func(ctx *baseContext.Context) {
	return func(ctx *baseContext.Context) {
		defer func() {
			if err := recover(); err != nil {
				if ctx.IsStopped() {
					return
				}

				var e error
				switch err.(type) {
				case error:
					e = baseError.WithStack(err.(error), 3)
				default:
					e = baseError.WithStack(errors.New(fmt.Sprint(err)), 3)
				}

				if ctx.Env != config.Production.String() {
					console := fmt.Sprintf("Recover: %s\n", reflect.TypeOf(err))
					console += fmt.Sprintf("%s\n", ctx.HandlerName())
					console += fmt.Sprintf("%+v", e)
					ctx.Application().Logger().Error(console)
				}

				ctx.Values().Set("error", e)

				ctx.StatusCode(500)
				ctx.StopExecution()
			}
		}()

		ctx.Next()
	}

}
