package api

import (
	"strings"

	"github.com/Unknwon/macaron"
	"github.com/raintank/raintank-apps/pkg/auth"
)

type Context struct {
	*macaron.Context
	*auth.SignedInUser
}

func GetContextHandler() macaron.Handler {
	return func(c *macaron.Context) {
		ctx := &Context{
			Context:      c,
			SignedInUser: &auth.SignedInUser{},
		}
		c.Map(ctx)
	}
}

func RequireAdmin() macaron.Handler {
	return func(ctx *Context) {
		if !ctx.IsAdmin {
			ctx.JSON(403, "Permision denied")
		}
	}
}

func Auth(adminKey string) macaron.Handler {
	return func(ctx *Context) {
		key := getApiKey(ctx)
		if key == "" {
			ctx.JSON(401, "Unauthorized")
			return
		}
		user, err := auth.Auth(adminKey, key)
		if err != nil {
			if err == auth.ErrInvalidApiKey {
				ctx.JSON(401, "Unauthorized")
				return
			}
			ctx.JSON(500, err)
			return
		}
		ctx.SignedInUser = user
	}
}

func getApiKey(c *Context) string {
	header := c.Req.Header.Get("Authorization")
	parts := strings.SplitN(header, " ", 2)
	if len(parts) == 2 && parts[0] == "Bearer" {
		key := parts[1]
		return key
	}

	return ""
}
