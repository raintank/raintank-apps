package api

import (
	"io/ioutil"

	"github.com/raintank/raintank-apps/tsdb/metric_publish"
	msg "github.com/raintank/raintank-metric/msg"
)

func Metrics(ctx *Context) {
	contentType := ctx.Req.Header.Get("Content-Type")
	switch contentType {
	case "rt-metric-binary":
		metricsBinary(ctx)
	case "application/json":
		metricsJson(ctx)
	default:
		ctx.JSON(400, "unknown content-type")
	}
}

func metricsJson(ctx *Context) {
	//TODO
	ctx.JSON(200, "ok")
}

func metricsBinary(ctx *Context) {
	defer ctx.Req.Request.Body.Close()
	if ctx.Req.Request.Body != nil {
		body, err := ioutil.ReadAll(ctx.Req.Request.Body)
		if err != nil {
			panic("unable to read requst body.")
		}
		ms, err := msg.MetricDataFromMsg(body)
		if err != nil {
			log.Errorf("event payload not metricData. %s", err.Error())
			ctx.JSON(500, err)
			return
		}

		err = ms.DecodeMetricData()
		if err != nil {
			log.Errorf("failed to unmarshal metricData. %s", err.Error())
			ctx.JSON(500, err)
			return
		}
		if !ctx.IsAdmin {
			for _, m := range ms.Metrics {
				m.OrgId = int(ctx.OrgId)
				m.SetId()
			}
		}

		err = metric_publish.Publish(ms.Metrics)
		if err != nil {
			log.Errorf("failed to publush metrics. %s", err)
			ctx.JSON(500, err)
			return
		}
		ctx.JSON(200, "ok")
		return
	}
	ctx.JSON(400, "no data included in request.")
}
