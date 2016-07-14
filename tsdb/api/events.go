package api

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/codeskyblue/go-uuid"
	"github.com/golang/snappy"
	"github.com/raintank/raintank-apps/tsdb/event_publish"
	"github.com/raintank/worldping-api/pkg/log"
	"gopkg.in/raintank/schema.v0"
	"gopkg.in/raintank/schema.v0/msg"
)

func Events(ctx *Context) {
	contentType := ctx.Req.Header.Get("Content-Type")
	switch contentType {
	case "rt-metric-binary":
		eventsBinary(ctx, false)
	case "rt-metric-binary-snappy":
		eventsBinary(ctx, true)
	case "application/json":
		eventsJson(ctx)
	default:
		ctx.JSON(400, fmt.Sprintf("unknown content-type: %s", contentType))
	}
}

func eventsJson(ctx *Context) {
	defer ctx.Req.Request.Body.Close()
	if ctx.Req.Request.Body != nil {
		body, err := ioutil.ReadAll(ctx.Req.Request.Body)
		if err != nil {
			log.Error(3, "unable to read requst body. %s", err)
		}
		event := new(schema.ProbeEvent)
		err = json.Unmarshal(body, event)
		if err != nil {
			ctx.JSON(400, fmt.Sprintf("unable to parse request body. %s", err))
			return
		}
		if !ctx.IsAdmin {
			event.OrgId = ctx.OrgId
		}

		u := uuid.NewUUID()
		event.Id = u.String()

		err = event_publish.Publish(event)
		if err != nil {
			log.Error(3, "failed to publush event. %s", err)
			ctx.JSON(500, err)
			return
		}
		ctx.JSON(200, "ok")
		return
	}
	ctx.JSON(400, "no data included in request.")
}

func eventsBinary(ctx *Context, compressed bool) {
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
		}
		ms, err := msg.ProbeEventFromMsg(body)
		if err != nil {
			log.Error(3, "event payload not Event. %s", err)
			ctx.JSON(500, err)
			return
		}

		err = ms.DecodeProbeEvent()
		if err != nil {
			log.Error(3, "failed to unmarshal EventData. %s", err)
			ctx.JSON(500, err)
			return
		}
		if !ctx.IsAdmin {
			ms.Event.OrgId = ctx.OrgId
		}
		u := uuid.NewUUID()
		ms.Event.Id = u.String()

		err = event_publish.Publish(ms.Event)
		if err != nil {
			log.Error(3, "failed to publush Event. %s", err)
			ctx.JSON(500, err)
			return
		}
		ctx.JSON(200, "ok")
		return
	}
	ctx.JSON(400, "no data included in request.")
}
