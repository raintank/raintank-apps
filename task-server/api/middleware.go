package api

import (
	"strings"

	"github.com/Unknwon/macaron"
	"github.com/raintank/raintank-apps/task-server/model"
)

type Context struct {
	*macaron.Context
	Owner   int64
	IsAdmin bool
}

func GetContextHandler() macaron.Handler {
	return func(c *macaron.Context) {
		ctx := &Context{
			Context: c,
			Owner:   0,
			IsAdmin: false,
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
		if key == adminKey {
			ctx.Owner = int64(1)
			ctx.IsAdmin = true
			return
		}
		//TODO: validate Key against Grafana.Net
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

func AgentQuota() macaron.Handler {
	return func(ctx *Context) {
		//check quotas for ctx.Owner
		return
	}
}

func TaskQuota() macaron.Handler {
	return func(ctx *Context, task model.TaskDTO) {
		// get quota for task.Metrics
		return
	}
}
