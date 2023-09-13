package session

import (
	"github.com/go-tron/config"
	"github.com/kataras/iris/v12/sessions"
	sessionRedis "github.com/kataras/iris/v12/sessions/sessiondb/redis"
	"github.com/redis/go-redis/v9"
)

func NewSessionsWithConfig(c *config.Config, client *redis.Client) *sessions.Sessions {
	db := sessionRedis.New(sessionRedis.Config{
		Prefix: c.GetString("application.name") + "-sess:",
		Driver: sessionRedis.GoRedis().SetClient(client),
	})
	sess := sessions.New(sessions.Config{
		Cookie:                      c.GetString("session.cookieName"),
		CookieSecureTLS:             true,
		AllowReclaim:                true,
		Expires:                     c.GetDuration("session.maxAge"),
		DisableSubdomainPersistence: true,
	})
	sess.UseDatabase(db)
	return sess
}
