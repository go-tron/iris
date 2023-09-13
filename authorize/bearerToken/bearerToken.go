package bearerToken

import (
	"github.com/go-tron/iris/baseContext"
	"github.com/kataras/iris/v12"
	"github.com/thoas/go-funk"
	"regexp"
	"strings"
)

type Authorize interface {
	Handler(*baseContext.Context, string) error
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{
		TokenProperty: "Authorization",
		TokenPrefix:   "Bearer ",
	}
}

func WithTokenProperty(val string) Option {
	return func(opts *Config) {
		opts.TokenProperty = val
	}
}
func WithTokenPrefix(val string) Option {
	return func(opts *Config) {
		opts.TokenPrefix = val
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
func WithInfoPaths(paths ...interface{}) Option {
	return func(opts *Config) {
		for _, path := range paths {
			opts.Paths = append(opts.Paths, PathConfig{path, LevelInfo})
		}
	}
}

func New(authorize Authorize, opts ...Option) *BearerToken {
	if authorize == nil {
		panic("Authorize 必须设置")
	}
	config := defaultConfig()
	for _, apply := range opts {
		apply(config)
	}
	return &BearerToken{
		Authorize: authorize,
		Config:    config,
	}
}

type Level int

const (
	LevelIgnore Level = iota
	LevelInfo
	LevelVerify
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
	TokenProperty string
	TokenPrefix   string
	Paths         []PathConfig
}

type BearerToken struct {
	Authorize
	*Config
	pathLevel []PathLevel
}

func (s *BearerToken) CheckPath(currPath string) Level {
	pathLevel := funk.Find(s.pathLevel, func(v PathLevel) bool {
		return v.Path == currPath
	})
	if pathLevel != nil {
		return pathLevel.(PathLevel).Level
	}

	for _, path := range s.Paths {
		switch v := (path.Name).(type) {
		case string:
			if v == currPath {
				s.pathLevel = append(s.pathLevel, PathLevel{
					Path:  currPath,
					Level: path.Level,
				})
				return path.Level
			}
		case *regexp.Regexp:
			if result := v.MatchString(currPath); result {
				s.pathLevel = append(s.pathLevel, PathLevel{
					Path:  currPath,
					Level: path.Level,
				})
				return path.Level
			}
		case func(string) bool:
			if result := v(currPath); result {
				s.pathLevel = append(s.pathLevel, PathLevel{
					Path:  currPath,
					Level: path.Level,
				})
				return path.Level
			}
		}
	}

	var defaultLevel = LevelVerify
	s.pathLevel = append(s.pathLevel, PathLevel{
		Path:  currPath,
		Level: defaultLevel,
	})
	return defaultLevel
}

func (s *BearerToken) Context(ctx *baseContext.Context) {
	path := ctx.Request().URL.Path
	level := s.CheckPath(path)
	if level == LevelIgnore {
		ctx.Next()
		return
	}

	token := ctx.GetHeader(s.TokenProperty)
	token = strings.Replace(token, s.TokenPrefix, "", 1)
	if err := s.Authorize.Handler(ctx, token); err != nil {
		if level == LevelInfo {
			ctx.Next()
			return
		}
		ctx.Error(err)
		return
	}
	ctx.Next()
}

func (s *BearerToken) Handler() iris.Handler {
	return baseContext.Handler(s.Context)
}
