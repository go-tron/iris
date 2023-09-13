package baseContext

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-playground/validator/v10"
	baseError "github.com/go-tron/base-error"
	"github.com/go-tron/config"
	localTime "github.com/go-tron/local-time"
	"github.com/go-tron/logger"
	"github.com/go-tron/types/jsonUtil"
	"github.com/go-tron/validate"
	"github.com/iris-contrib/schema"
	"github.com/kataras/iris/v12"
	"github.com/kataras/iris/v12/sessions"
	"reflect"
	"strings"
)

var (
	ErrorSystem     = baseError.SystemFactory("100")
	ErrorHandler    = baseError.SystemFactory("101", "routing failed:{}")
	ErrorSession    = baseError.SystemFactory("102")
	ErrorReadParams = baseError.Factory("1001", "params read failed:{}")
	ErrorValidation = baseError.Factory("1002", "params validate failed:{}")
)

type Logger interface {
	GetLogger() logger.Logger
	Handler() iris.Handler
	Log(*Context)
}

type Response interface {
	New(code string, msg string, data ...interface{}) Response
	Success(data ...interface{}) Response
	Error(code string, msg string, data ...interface{}) Response
	WithCode(code string) Response
	WithMessage(message string) Response
	WithData(data ...interface{}) Response
	WithSystem() Response
	WithChain(chain string) Response
	WithRid(rid string) Response
	ContentType() string
	Content() interface{}
}

type Option func(*Context)

func New(env string, logger Logger, opts ...Option) {
	baseContext = &Context{
		Env:    env,
		Logger: logger,
	}
	for _, apply := range opts {
		apply(baseContext)
	}
}

func WithApplicationName(val string) Option {
	return func(opts *Context) {
		opts.ApplicationName = val
	}
}

func WithInternal(val bool) Option {
	return func(opts *Context) {
		opts.Internal = val
	}
}

func WithResponse(val Response) Option {
	return func(opts *Context) {
		opts.Response = val
	}
}

func WithViewError(val string) Option {
	return func(opts *Context) {
		opts.ViewError = val
	}
}

func WithSystemErrorCode(val string) Option {
	return func(opts *Context) {
		opts.SystemErrorCode = val
	}
}

type Context struct {
	iris.Context
	Env             string
	ApplicationName string
	Internal        bool
	Logger          Logger
	Response        Response
	ViewError       string
	SystemErrorCode string
}

const irisSessionContextKey = "iris.session"

func (ctx *Context) GetSession() *sessions.Session {
	if v := ctx.Values().Get(irisSessionContextKey); v != nil {
		if sess, ok := v.(*sessions.Session); ok {
			return sess
		}
	}
	return nil
}

func UnmarshalJSON(data []byte, v interface{}) error {
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

func (ctx *Context) ReadJSONUseNumber(p interface{}) error {
	if err := ctx.UnmarshalBody(p, iris.UnmarshalerFunc(jsonUtil.UnmarshalUseNumber)); err != nil {
		return err
	}
	return nil
}

func (ctx *Context) JSONReqBody(p interface{}) error {
	if err := ctx.UnmarshalBody(p, iris.UnmarshalerFunc(UnmarshalJSON)); err != nil {
		return err
	}
	if reflect.TypeOf(p).Kind() == reflect.Struct || (reflect.TypeOf(p).Kind() == reflect.Ptr && reflect.TypeOf(p).Elem().Kind() == reflect.Struct) {
		if err := validate.Validate.Struct(p); err != nil {
			return ErrorValidation(err)
		}
	}
	return nil
}

func (ctx *Context) JSONReqForm(p interface{}) error {
	if err := ctx.ReadForm(p); err != nil {
		return err
	}
	if reflect.TypeOf(p).Kind() == reflect.Struct || (reflect.TypeOf(p).Kind() == reflect.Ptr && reflect.TypeOf(p).Elem().Kind() == reflect.Struct) {
		if err := validate.Validate.Struct(p); err != nil {
			return ErrorValidation(err)
		}
	}
	return nil
}

func (ctx *Context) NewResponse(code string, msg string, data ...interface{}) Response {
	return ctx.Response.New(code, msg, data...)
}
func (ctx *Context) NewSuccess(data ...interface{}) Response {
	return ctx.Response.Success(data...)
}
func (ctx *Context) Success(data ...interface{}) {
	ctx.StopExecution()
	resp := ctx.Response.Success(data...)
	if requestId := ctx.Values().GetString("requestId"); requestId != "" {
		resp.WithRid(requestId)
	}
	if resp.ContentType() == "text" {
		ctx.Text(resp.Content().(string))
	} else if resp.ContentType() == "binary" {
		ctx.Binary(resp.Content().([]byte))
	} else {
		ctx.JSON(resp.Content())
	}
}

func (ctx *Context) BaseError(err error) (e *baseError.Error) {
	ctx.Values().Set("error", err)
	var errorType = reflect.TypeOf(err).String()
	//baseError
	if errorType == "*baseError.Error" {
		e = err.(*baseError.Error)
		if e.System || e.Stack() != nil {
			if ctx.Env != config.Production.String() {
				console := fmt.Sprintf("Error: %s\n", errorType)
				console += fmt.Sprintf("%+v", err)
				ctx.Application().Logger().Error(console)
			}
		}
		return e
	}

	//参数校验失败
	if errorType == "*json.SyntaxError" || errorType == "validator.ValidationErrors" || errorType == "schema.MultiError" {
		if errorType == "*json.SyntaxError" {
			e = ErrorReadParams(err.Error())
		} else if errorType == "validator.ValidationErrors" {
			e = ErrorValidation(err.(validator.ValidationErrors).Translate(validate.Validate.Trans))
		} else {
			emap := err.(schema.MultiError)
			var errList = make([]string, 0)
			for k, _ := range emap {
				errList = append(errList, k)
			}
			e = ErrorReadParams(strings.Join(errList, ","))
		}
		return e
	}

	//其他错误
	if ctx.Env != config.Production.String() {
		console := fmt.Sprintf("Error: %s\n", errorType)
		console += fmt.Sprintf("%+v", err)
		ctx.Application().Logger().Error(console)
	}

	var message = err.Error()
	if ctx.SystemErrorCode != "" {
		return baseError.System(ctx.SystemErrorCode, message)
	} else {
		return ErrorSystem(message)
	}
}

func (ctx *Context) Error(err error, data ...interface{}) {
	ctx.StopExecution()
	e := ctx.BaseError(err)
	message := e.Msg
	if e.System && ctx.Env == config.Production.String() && !ctx.Internal {
		message = "system error"
	}
	resp := ctx.Response.Error(e.Code, message, data...)
	if requestId := ctx.Values().GetString("requestId"); requestId != "" {
		resp.WithRid(requestId)
	}
	if e.System {
		resp.WithSystem()
	}
	if e.Chain != "" {
		resp.WithChain(e.Chain)
	}
	if resp.ContentType() == "text" {
		ctx.Text(resp.Content().(string))
	} else if resp.ContentType() == "binary" {
		ctx.Binary(resp.Content().([]byte))
	} else {
		ctx.JSON(resp.Content())
	}
}

func (ctx *Context) ErrorView(err error, data ...interface{}) {
	ctx.StopExecution()
	e := ctx.BaseError(err)
	message := e.Msg
	if e.System && ctx.Env == config.Production.String() && !ctx.Internal {
		message = "system error"
	}
	if requestId := ctx.Values().GetString("requestId"); requestId != "" {
		message += " rid:" + requestId
	}
	ctx.ViewData("Message", fmt.Sprintf("[%s]%s", e.Code, message))
	ctx.View(ctx.ViewError)
}

func (ctx *Context) GetIP() string {
	ip := ctx.RemoteAddr()
	if ctx.GetHeader("X-REAL-IP") != "" {
		ip = ctx.GetHeader("X-REAL-IP")
	}
	return ip
}

func (ctx *Context) GetRequestURI() string {
	scheme := ctx.Request().URL.Scheme
	if scheme == "" {
		if ctx.Request().TLS != nil {
			scheme = "https:"
		} else {
			scheme = "http:"
		}
	}
	return scheme + "//" + ctx.Host() + ctx.Request().RequestURI
}

func (ctx *Context) GetQuery(name string) string {
	var m = make(map[string]string)
	arr := strings.Split(ctx.Request().URL.RawQuery, "&")
	for _, temp := range arr {
		index := strings.Index(temp, "=")
		if index == -1 {
			continue
		}
		key := temp[0:index]
		value := temp[index+1:]
		m[key] = value
	}
	return m[name]
}

func (ctx *Context) LogField(key string, value interface{}) *logger.Field {
	return ctx.Logger.GetLogger().Field(key, value)
}

func (ctx *Context) AddLogField(key string, value interface{}) {
	logFields := ctx.GetLogFields()
	logFields = append(logFields, ctx.Logger.GetLogger().Field(key, value))
	ctx.Values().Set("logFields", logFields)
}

func (ctx *Context) AddLogFields(fields ...*logger.Field) {
	logFields := ctx.GetLogFields()
	logFields = append(logFields, fields...)
	ctx.Values().Set("logFields", logFields)
}

func (ctx *Context) GetLogFields() []*logger.Field {
	ctxLog := ctx.Values().Get("logFields")
	if ctxLog != nil {
		logFields, ok := ctxLog.([]*logger.Field)
		if ok {
			return logFields
		}
	}
	var logFields = make([]*logger.Field, 0)
	ctx.Values().Set("logFields", logFields)
	return logFields
}

func (ctx *Context) AddLogSessionKeys(keys ...string) {
	ctx.Values().Set("logSessionKeys", keys)
}

func (ctx *Context) GetLogSessionKeys() []string {
	keys := ctx.Values().Get("logSessionKeys")
	if keys != nil {
		v, ok := keys.([]string)
		if ok {
			return v
		}
	}
	return nil
}

func (ctx *Context) AddLogContextKeys(keys ...string) {
	ctx.Values().Set("logContextKeys", keys)
}

func (ctx *Context) GetLogContextKeys() []string {
	keys := ctx.Values().Get("logContextKeys")
	if keys != nil {
		v, ok := keys.([]string)
		if ok {
			return v
		}
	}
	return nil
}

func (ctx *Context) GetTraceCtx() context.Context {
	if v := ctx.Values().Get("traceCtx"); v != nil {
		traceCtx, ok := v.(context.Context)
		if ok {
			return traceCtx
		}
	}
	return ctx
}

func (ctx *Context) SetTraceCtx(traceCtx context.Context) {
	ctx.Values().Set("traceCtx", traceCtx)
}

var formDecoder *schema.Decoder

func (ctx *Context) ReadForm(p interface{}) error {
	values := ctx.FormValues()
	if len(values) == 0 {
		return nil
	}

	if reflect.TypeOf(p).Kind() == reflect.Map || (reflect.TypeOf(p).Kind() == reflect.Ptr && reflect.TypeOf(p).Elem().Kind() == reflect.Map) {
		pV := reflect.ValueOf(p)
		if pV.Kind() == reflect.Ptr {
			pV = pV.Elem()
		}
		if pV.IsZero() {
			pV.Set(reflect.MakeMap(pV.Type()))
		}
		for k, v := range values {
			if len(v) == 1 {
				pV.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v[0]))
			} else {
				pV.SetMapIndex(reflect.ValueOf(k), reflect.ValueOf(v))
			}
		}
		return nil
	}

	if formDecoder == nil {
		formDecoder = schema.NewDecoder()
		formDecoder.IgnoreUnknownKeys(true)
		formDecoder.RegisterConverter(localTime.Time{}, func(value string) reflect.Value {
			now, err := localTime.ParseLocal(value)
			if err != nil {
				return reflect.Value{}
			}
			return reflect.ValueOf(now)
		})
	}

	return formDecoder.Decode(p, values)
}
