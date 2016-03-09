package api

import (
	"github.com/raintank/raintank-apps/tsdb/graphite"
)

func GraphiteProxy(c *Context) {
	proxyPath := c.Params("*")
	proxy := graphite.Proxy(c.Owner, proxyPath, c.Req.Request)
	proxy.ServeHTTP(c.RW(), c.Req.Request)
}
