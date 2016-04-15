package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/raintank-apps/tsdb/metric_publish"
	msg "github.com/raintank/raintank-metric/msg"
	"github.com/raintank/raintank-metric/schema"
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
	defer ctx.Req.Request.Body.Close()
	if ctx.Req.Request.Body != nil {
		body, err := ioutil.ReadAll(ctx.Req.Request.Body)
		if err != nil {
			log.Error(3, "unable to read requst body. %s", err)
		}
		metrics := make([]*schema.MetricData, 0)
		err = json.Unmarshal(body, &metrics)
		if err != nil {
			ctx.JSON(400, fmt.Sprintf("unable to parse request body. %s", err))
			return
		}
		if !ctx.IsAdmin {
			for _, m := range metrics {
				m.OrgId = int(ctx.OrgId)
				m.SetId()
			}
		}

		err = metric_publish.Publish(metrics)
		if err != nil {
			log.Error(3, "failed to publush metrics. %s", err)
			ctx.JSON(500, err)
			return
		}
		ctx.JSON(200, "ok")
		return
	}
	ctx.JSON(400, "no data included in request.")
}

func metricsBinary(ctx *Context) {
	defer ctx.Req.Request.Body.Close()
	if ctx.Req.Request.Body != nil {
		body, err := ioutil.ReadAll(ctx.Req.Request.Body)
		if err != nil {
			log.Error(3, "unable to read requst body. %s", err)
			ctx.JSON(500, err)
			return
		}
		ms, err := msg.MetricDataFromMsg(body)
		if err != nil {
			log.Error(3, "event payload not metricData. %s", err)
			ctx.JSON(500, err)
			return
		}

		err = ms.DecodeMetricData()
		if err != nil {
			log.Error(3, "failed to unmarshal metricData. %s", err)
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
			log.Error(3, "failed to publush metrics. %s", err)
			ctx.JSON(500, err)
			return
		}
		ctx.JSON(200, "ok")
		return
	}
	ctx.JSON(400, "no data included in request.")
}
