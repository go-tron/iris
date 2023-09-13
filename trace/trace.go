package trace

import (
	"context"
	baseError "github.com/go-tron/base-error"
	"github.com/go-tron/iris/baseContext"
	"github.com/kataras/iris/v12"
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"
	"github.com/opentracing/opentracing-go/log"
	uuid "github.com/satori/go.uuid"
	"github.com/uber/jaeger-client-go"
	"reflect"
)

func New() iris.Handler {
	return baseContext.Handler(Trace)
}

func Trace(ctx *baseContext.Context) {
	//fmt.Println("Header", ctx.Request().Header)

	requestId := ctx.GetHeader("x-request-id")
	if requestId == "" {
		requestId = uuid.NewV4().String()
	}

	//fmt.Println("requestId", requestId)
	ctx.Values().Set("requestId", requestId)
	//r := ctx.Request()
	traceCtx := context.WithValue(context.Background(), "x-request-id", requestId)

	var opts []opentracing.StartSpanOption
	if tracer := opentracing.GlobalTracer(); tracer != nil {
		spanCtx, _ := tracer.Extract(opentracing.HTTPHeaders, opentracing.HTTPHeadersCarrier(ctx.Request().Header))
		if spanCtx != nil {
			opts = append(opts, ext.RPCServerOption(spanCtx))
		}
		span := tracer.StartSpan(ctx.Request().URL.Path+":S:", opts...)
		defer func() {
			if err := ctx.Values().Get("error"); err != nil {
				if reflect.TypeOf(err).String() == "*baseError.Error" && !err.(*baseError.Error).System {
					span.LogFields(log.String("error", err.(error).Error()))
				} else {
					ext.LogError(span, err.(error))
				}
			}
			span.Finish()
		}()

		span.SetTag("x-request-id", requestId)
		ctx.Values().Set("traceId", span.Context().(jaeger.SpanContext).TraceID())

		//fmt.Println("traceId", span.Context().(jaeger.SpanContext).TraceID())
		//fmt.Println("parentId", span.Context().(jaeger.SpanContext).ParentID())
		//fmt.Println("spanId", span.Context().(jaeger.SpanContext).SpanID())
		traceCtx = opentracing.ContextWithSpan(context.WithValue(context.Background(), "x-request-id", requestId), span)
	}

	ctx.SetTraceCtx(traceCtx)
	//ctx.ResetRequest(r.WithContext(traceCtx))
	ctx.Next()
}
