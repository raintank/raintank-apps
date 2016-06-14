package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/codeskyblue/go-uuid"
	"github.com/raintank/raintank-apps/tsdb/event_publish"
	msg "github.com/raintank/raintank-metric/msg"
	"github.com/raintank/raintank-metric/schema"
	"github.com/raintank/worldping-api/pkg/log"
)

func Events(ctx *Context) {
	contentType := ctx.Req.Header.Get("Content-Type")
	switch contentType {
	case "rt-metric-binary":
		eventsBinary(ctx)
	case "application/json":
		eventsJson(ctx)
	default:
		ctx.JSON(400, "unknown content-type")
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

func eventsBinary(ctx *Context) {
	defer ctx.Req.Request.Body.Close()
	if ctx.Req.Request.Body != nil {
		body, err := ioutil.ReadAll(ctx.Req.Request.Body)
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
