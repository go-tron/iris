package health

import (
	localTime "github.com/go-tron/local-time"
	"github.com/kataras/iris/v12"
)

func defaultConfig() *Config {
	return &Config{
		Path: "/health/check",
	}
}

type Config struct {
	Path     string
	Handlers []iris.Handler
}

type Option func(*Config)

func WithPath(path string) Option {
	return func(opts *Config) {
		opts.Path = path
	}
}

func WithHandlers(handlers ...iris.Handler) Option {
	return func(opts *Config) {
		opts.Handlers = append(opts.Handlers, handlers...)
	}
}

func New(app *iris.Application, opts ...Option) {
	config := defaultConfig()
	for _, apply := range opts {
		apply(config)
	}

	config.Handlers = append(config.Handlers, func(ctx iris.Context) {
		ctx.Text("check at:" + localTime.Now().String())
	})

	app.Post(config.Path, config.Handlers...)
}
