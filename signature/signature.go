package signature

import (
	"encoding/json"
	"github.com/go-tron/base-error"
	"github.com/go-tron/iris/baseContext"
	"github.com/kataras/iris/v12"
	"github.com/thoas/go-funk"
	"regexp"
	"strconv"
	"time"
)

var (
	ErrorNoTimestamp       = baseError.Factory("3001", "{}")
	ErrorTimestampAfterNow = baseError.Factory("3002", "{}time can't after now")
	ErrorTimestampExpired  = baseError.Factory("3003", "{}expired (validity{})")
)

type Signer interface {
	Verify(map[string]interface{}) error
}

type Option func(*Config)

func defaultConfig() *Config {
	return &Config{}
}
func New(signer Signer, opts ...Option) *Signature {
	if signer == nil {
		panic("signer 必须设置")
	}
	config := defaultConfig()
	for _, apply := range opts {
		apply(config)
	}
	return &Signature{
		Signer: signer,
		Config: config,
	}
}

func WithTimestamp(timestamp *Timestamp) Option {
	return func(opts *Config) {
		if timestamp == nil {
			panic("timestamp 必须设置")
		}
		if timestamp.Duration == 0 {
			panic("必须设置duration")
		}
		if timestamp.Property == "" {
			timestamp.Property = "timestamp"
		}
		opts.Timestamp = timestamp
	}
}
func WithBodyType(bodyType BodyType) Option {
	return func(opts *Config) {
		opts.BodyType = bodyType
	}
}
func WithPath(path string, level Level) Option {
	return func(opts *Config) {
		opts.Paths = append(opts.Paths, PathConfig{path, level})
	}
}
func WithPaths(paths ...PathConfig) Option {
	return func(opts *Config) {
		opts.Paths = append(opts.Paths, paths...)
	}
}
func WithIgnorePaths(paths ...string) Option {
	return func(opts *Config) {
		for _, path := range paths {
			opts.Paths = append(opts.Paths, PathConfig{path, LevelIgnore})
		}
	}
}

type Level int

const (
	LevelUnset Level = iota
	LevelIgnore
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

type BodyType int

const (
	BodyTypeJSON BodyType = iota
	BodyTypeForm
	BodyTypeQuery
)

type Timestamp struct {
	Property string
	Duration time.Duration
	Unit     time.Duration
}

type Config struct {
	BodyType  BodyType
	Timestamp *Timestamp
	Paths     []PathConfig
}

type Signature struct {
	Signer
	*Config
	pathLevel []PathLevel
}

func (s *Signature) CheckPath(currPath string) Level {

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
		}
	}

	var defaultLevel = LevelVerify
	s.pathLevel = append(s.pathLevel, PathLevel{
		Path:  currPath,
		Level: defaultLevel,
	})
	return defaultLevel
}

func (s *Signature) Context(ctx *baseContext.Context) {
	level := s.CheckPath(ctx.Request().URL.Path)
	if level == LevelIgnore {
		ctx.Next()
		return
	}

	var params map[string]interface{}
	if s.Config.BodyType == BodyTypeJSON {
		if err := ctx.ReadJSONUseNumber(&params); err != nil {
			ctx.Error(err)
			return
		}
	} else if s.Config.BodyType == BodyTypeForm {
		if err := ctx.ReadForm(&params); err != nil {
			ctx.Error(err)
			return
		}
	}

	if s.Config.Timestamp != nil {
		timestamp := params[s.Config.Timestamp.Property]
		var t int64
		switch v := timestamp.(type) {
		case string:
			t, _ = strconv.ParseInt(v, 10, 64)
		case json.Number:
			t, _ = v.Int64()
		case float64:
			t = int64(v)
		default:
			ctx.Error(ErrorNoTimestamp(s.Config.Timestamp.Property))
			return
		}

		tm := time.Unix(t/int64(time.Second/s.Config.Timestamp.Unit), 0)
		if time.Until(tm) > 10*time.Second {
			ctx.Error(ErrorTimestampAfterNow(s.Config.Timestamp.Property))
			return
		}
		if time.Since(tm) > s.Config.Timestamp.Duration {
			ctx.Error(ErrorTimestampExpired(s.Config.Timestamp.Property, s.Config.Timestamp.Duration))
			return
		}
	}

	if err := s.Signer.Verify(params); err != nil {
		ctx.Error(err)
		return
	}
	ctx.Next()
}

func (s *Signature) Handler() iris.Handler {
	return baseContext.Handler(s.Context)
}
