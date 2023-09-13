package requestLogger

import (
	baseError "github.com/go-tron/base-error"
	"github.com/go-tron/iris/baseContext"
	"github.com/go-tron/logger"
	"github.com/kataras/iris/v12"
	"github.com/thoas/go-funk"
	"reflect"
	"regexp"
	"time"
)

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		IP:       true,
		Query:    true,
		Body:     true,
		Response: false,
	}
}

func New(logger logger.Logger, opts ...Option) *RequestLogger {
	if logger == nil {
		panic("logger 必须设置")
	}

	config := defaultConfig()
	for _, apply := range opts {
		apply(config)
	}
	return &RequestLogger{
		logger: logger,
		Config: config,
	}
}

func WithIP(val bool) Option {
	return func(opts *Config) {
		opts.IP = val
	}
}
func WithQuery(val bool) Option {
	return func(opts *Config) {
		opts.Query = val
	}
}
func WithBody(val bool) Option {
	return func(opts *Config) {
		opts.Body = val
	}
}
func WithUserAgent(val bool) Option {
	return func(opts *Config) {
		opts.UserAgent = val
	}
}
func WithResponse(val bool) Option {
	return func(opts *Config) {
		opts.Response = val
	}
}
func WithContextKeys(val ...string) Option {
	return func(opts *Config) {
		opts.ContextKeys = append(opts.ContextKeys, val...)
	}
}
func WithHeaderKeys(val ...string) Option {
	return func(opts *Config) {
		opts.HeaderKeys = append(opts.HeaderKeys, val...)
	}
}
func WithSessionKeys(val ...string) Option {
	return func(opts *Config) {
		opts.SessionKeys = append(opts.SessionKeys, val...)
	}
}
func WithPath(path interface{}, level Level) Option {
	return func(opts *Config) {
		opts.Paths = append(opts.Paths, PathConfig{path, level})
	}
}
func WithPaths(paths ...PathConfig) Option {
	return func(opts *Config) {
		opts.Paths = append(opts.Paths, paths...)
	}
}
func WithIgnorePaths(paths ...interface{}) Option {
	return func(opts *Config) {
		for _, path := range paths {
			opts.Paths = append(opts.Paths, PathConfig{path, LevelIgnore})
		}
	}
}
func WithNoResponsePaths(paths ...interface{}) Option {
	return func(opts *Config) {
		for _, path := range paths {
			opts.Paths = append(opts.Paths, PathConfig{path, LevelNoResponse})
		}
	}
}
func WithResponsePaths(paths ...interface{}) Option {
	return func(opts *Config) {
		for _, path := range paths {
			opts.Paths = append(opts.Paths, PathConfig{path, LevelResponse})
		}
	}
}

type Level int

const (
	LevelIgnore Level = iota
	LevelNoResponse
	LevelResponse
)

type PathLevel struct {
	Path  string
	Level Level
}

type PathConfig struct {
	Name  interface{}
	Level Level
}

type Config struct {
	IP          bool
	Query       bool
	Body        bool
	Response    bool
	UserAgent   bool
	ContextKeys []string
	HeaderKeys  []string
	SessionKeys []string
	Paths       []PathConfig
}

type RequestLogger struct {
	logger logger.Logger
	*Config
	pathLevel []PathLevel
}

func (l *RequestLogger) GetLogger() logger.Logger {
	return l.logger
}

func (l *RequestLogger) Context(ctx *baseContext.Context) {
	level := l.CheckPath(ctx.Request().URL.Path)
	if level == LevelIgnore {
		ctx.Next()
		return
	}

	ctx.Values().Set("startTime", time.Now())
	if level == LevelResponse {
		ctx.Record()
	}
	ctx.Next()
	l.Log(ctx)
}
func (l *RequestLogger) Handler() iris.Handler {
	return baseContext.Handler(l.Context)
}

func (l *RequestLogger) CheckPath(currPath string) Level {

	pathLevel := funk.Find(l.pathLevel, func(v PathLevel) bool {
		return v.Path == currPath
	})
	if pathLevel != nil {
		return pathLevel.(PathLevel).Level
	}

	for _, path := range l.Paths {
		switch v := (path.Name).(type) {
		case string:
			if v == currPath {
				l.pathLevel = append(l.pathLevel, PathLevel{
					Path:  currPath,
					Level: path.Level,
				})
				return path.Level
			}
		case *regexp.Regexp:
			if result := v.MatchString(currPath); result {
				l.pathLevel = append(l.pathLevel, PathLevel{
					Path:  currPath,
					Level: path.Level,
				})
				return path.Level
			}
		}
	}

	var defaultLevel Level
	if l.Response {
		defaultLevel = LevelResponse
	} else {
		defaultLevel = LevelNoResponse
	}

	l.pathLevel = append(l.pathLevel, PathLevel{
		Path:  currPath,
		Level: defaultLevel,
	})
	return defaultLevel
}

func (l *RequestLogger) Log(ctx *baseContext.Context) {

	startTime := ctx.Values().Get("startTime")
	var latency int64
	if startTime != nil {
		latency = time.Since(startTime.(time.Time)).Milliseconds()
	}

	requestBody, _ := ctx.GetBody()

	fields := []*logger.Field{
		l.logger.Field("time", startTime),
		l.logger.Field("method", ctx.Request().Method),
		l.logger.Field("host", ctx.Request().Host),
		l.logger.Field("uri", ctx.Request().RequestURI),
		l.logger.Field("path", ctx.Request().URL.Path),
		l.logger.Field("latency", latency),
		l.logger.Field("status", ctx.ResponseWriter().StatusCode()),
	}

	if l.IP {
		fields = append(fields, l.logger.Field("ip", ctx.GetIP()))
	}
	if l.Query {
		fields = append(fields, l.logger.Field("query", ctx.Request().URL.RawQuery))
	}
	if l.Body {
		fields = append(fields, l.logger.Field("body", requestBody))
	}
	if l.UserAgent {
		fields = append(fields, l.logger.Field("user-agent", ctx.GetHeader("user-agent")))
	}
	if l.CheckPath(ctx.Request().URL.Path) == LevelResponse {
		fields = append(fields, l.logger.Field("response", ctx.Recorder().Body()))
	}

	if headerKeys := l.HeaderKeys; len(headerKeys) > 0 {
		for _, key := range headerKeys {
			if value := ctx.GetHeader(key); value != "" {
				fields = append(fields, l.logger.Field(key, value))
			}
		}
	}

	if ctxKeys := l.ContextKeys; len(ctxKeys) > 0 {
		for _, key := range ctxKeys {
			if value := ctx.Values().Get(key); value != nil {
				fields = append(fields, l.logger.Field(key, value))
			}
		}
	}

	if logFields := ctx.GetLogFields(); len(logFields) > 0 {
		fields = append(fields, logFields...)
	}
	if contextKeys := ctx.GetLogContextKeys(); len(contextKeys) > 0 {
		for _, key := range contextKeys {
			if value := ctx.Values().Get(key); value != nil {
				fields = append(fields, l.logger.Field(key, value))
			}
		}
	}

	if session := ctx.GetSession(); session != nil {
		fields = append(fields, l.logger.Field("session_id", session.ID()))
		if sessionKeys := append(l.SessionKeys, ctx.GetLogSessionKeys()...); len(sessionKeys) > 0 {
			for _, key := range sessionKeys {
				if value := session.Get(key); value != nil {
					fields = append(fields, l.logger.Field(key, value))
				}
			}
		}
	}

	if requestId := ctx.Values().GetString("requestId"); requestId != "" {
		fields = append(fields, l.logger.Field("request_id", requestId))
	}
	if traceId := ctx.Values().GetString("traceId"); traceId != "" {
		fields = append(fields, l.logger.Field("trace_id", traceId))
	}

	ctxErr := ctx.Values().Get("error")
	if ctxErr != nil {
		fields = append(fields, l.logger.Field("error", ctxErr))
		level := "error"
		if reflect.TypeOf(ctxErr).String() == "*baseError.Error" {
			e := ctxErr.(*baseError.Error)
			if !e.System {
				level = "warn"
			}
			if e.Chain != "" {
				fields = append(fields, l.logger.Field("error_chain", e.Chain))
			}
		}
		if level == "warn" {
			l.logger.Warn("", fields...)
		} else {
			l.logger.Error("", fields...)
		}
	} else {
		l.logger.Info("", fields...)
	}
}
