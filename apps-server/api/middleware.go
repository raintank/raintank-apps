package api

import (
	"strings"

	"github.com/Unknwon/macaron"
)

type Context struct {
	*macaron.Context
	Owner int64
}

func GetContextHandler() macaron.Handler {
	return func(c *macaron.Context) {
		ctx := &Context{
			Context: c,
			Owner:   0,
		}
		c.Map(ctx)
	}
}

func Auth(adminKey string) macaron.Handler {
	return func(ctx *Context) {
		key := getApiKey(ctx)
		if key == "" {
			ctx.JSON(403, "Permission denied")
			return
		}
		if key == adminKey {
			ctx.Owner = int64(1)
			return
		}
		// validate Key
		ctx.Owner = int64(2)
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
