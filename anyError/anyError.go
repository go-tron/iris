package anyError

import (
	"errors"
	"github.com/go-tron/iris/baseContext"
	"github.com/kataras/iris/v12"
	"net/http"
	"strconv"
)

func New() iris.Handler {
	return baseContext.Handler(AnyError)
}

func AnyError(ctx *baseContext.Context) {
	text := http.StatusText(ctx.GetStatusCode())

	if ctx.Values().Get("error") == nil {
		ctx.Values().Set("error", errors.New(text))
	}
	if ctx.GetContentTypeRequested() != "" {
		ctx.JSON(ctx.Response.Error(strconv.Itoa(ctx.GetStatusCode()), text))
	} else {
		ctx.WriteString(text)
	}
	ctx.Logger.Log(ctx)
}
