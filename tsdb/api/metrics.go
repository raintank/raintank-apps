package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/golang/snappy"
	"github.com/raintank/raintank-apps/tsdb/metric_publish"
	"github.com/raintank/worldping-api/pkg/log"
	"gopkg.in/raintank/schema.v0"
	"gopkg.in/raintank/schema.v0/msg"
)

func Metrics(ctx *Context) {
	contentType := ctx.Req.Header.Get("Content-Type")
	switch contentType {
	case "rt-metric-binary":
		metricsBinary(ctx, false)
	case "rt-metric-binary-snappy":
		metricsBinary(ctx, true)
	case "application/json":
		metricsJson(ctx)
	default:
		ctx.JSON(400, fmt.Sprintf("unknown content-type: %s", contentType))
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

func metricsBinary(ctx *Context, compressed bool) {
	var body io.ReadCloser
	if compressed {
		body = ioutil.NopCloser(snappy.NewReader(ctx.Req.Request.Body))
	} else {
		body = ctx.Req.Request.Body
	}
	defer body.Close()

	if ctx.Req.Request.Body != nil {
		body, err := ioutil.ReadAll(body)
		if err != nil {
			log.Error(3, "unable to read requst body. %s", err)
			ctx.JSON(500, err)
			return
		}
		metricData := new(msg.MetricData)
		err = metricData.InitFromMsg(body)
		if err != nil {
			log.Error(3, "payload not metricData. %s", err)
			ctx.JSON(500, err)
			return
		}

		err = metricData.DecodeMetricData()
		if err != nil {
			log.Error(3, "failed to unmarshal metricData. %s", err)
			ctx.JSON(500, err)
			return
		}
		if !ctx.IsAdmin {
			for _, m := range metricData.Metrics {
				m.OrgId = int(ctx.OrgId)
				m.SetId()
			}
		}

		err = metric_publish.Publish(metricData.Metrics)
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
