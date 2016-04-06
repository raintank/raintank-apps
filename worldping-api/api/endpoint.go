package api

import (
	"time"

	"github.com/grafana/grafana/pkg/log"
	"github.com/raintank/met"
	"github.com/raintank/raintank-apps/worldping-api/endpoint_discovery"
	"github.com/raintank/raintank-apps/worldping-api/model"
	"github.com/raintank/raintank-apps/worldping-api/sqlstore"
)

var (
	//counters
	endpointAddOk      met.Count
	endpointUpdateOk   met.Count
	endpointDeleteOk   met.Count
	endpointAddFail    met.Count
	endpointUpdateFail met.Count
	endpointDeleteFail met.Count

	//timers
	endpointAddDuration    met.Timer
	endpointUpdateDuration met.Timer
	endpointDeleteDuration met.Timer
)

func InitEndpointMetrics(stats met.Backend) {
	endpointAddOk = stats.NewCount("endpoint_add.ok")
	endpointAddFail = stats.NewCount("endpoint_add.fail")
	endpointUpdateOk = stats.NewCount("endpoint_update.ok")
	endpointUpdateFail = stats.NewCount("endpoint_update.fail")
	endpointDeleteOk = stats.NewCount("endpoint_delete.ok")
	endpointDeleteFail = stats.NewCount("endpoint_delete.fail")

	endpointAddDuration = stats.NewTimer("endpoint_add_duration", 0)
	endpointUpdateDuration = stats.NewTimer("endpoint_update_duration", 0)
	endpointDeleteDuration = stats.NewTimer("endpoint_delete_duration", 0)
}

func GetEndpoints(ctx *Context, query model.GetEndpointsQuery) {
	query.OrgId = ctx.OrgId
	endpoints, err := sqlstore.GetEndpoints(&query)
	if err != nil {
		log.Error(3, "api.GetEndpoints failed. %s", err)
		ctx.JSON(500, err)
		return
	}
	ctx.JSON(200, endpoints)
}

func GetEndpointById(ctx *Context) {
	id := ctx.ParamsInt64(":id")
	endpoint, err := sqlstore.GetEndpointById(id, ctx.OrgId)
	if err != nil {
		log.Error(3, "api.GetEndpointById failed. %s", err)
		ctx.JSON(500, err)
		return
	}
	if endpoint == nil {
		ctx.JSON(404, "Endpoint not found")
		return
	}
	ctx.JSON(200, endpoint)
}

func AddEndpoint(ctx *Context, e model.EndpointDTO) {
	pre := time.Now()
	e.OrgId = ctx.OrgId
	err := sqlstore.AddEndpoint(&e)
	if err != nil {
		log.Error(3, "api.AddEndpoint failed. %s", err)
		ctx.JSON(500, err)
		endpointAddFail.Inc(1)
		return
	}
	endpointAddDuration.Value(time.Now().Sub(pre))
	endpointAddOk.Inc(1)
	ctx.JSON(200, e)
}

func UpdateEndpoint(ctx *Context, e model.EndpointDTO) {
	pre := time.Now()
	e.OrgId = ctx.OrgId
	err := sqlstore.UpdateEndpoint(&e)
	if err != nil {
		log.Error(3, "api.UpdateEndpoint failed. %s", err)
		ctx.JSON(500, err)
		endpointUpdateFail.Inc(1)
		return
	}
	endpointUpdateDuration.Value(time.Now().Sub(pre))
	endpointUpdateOk.Inc(1)
	ctx.JSON(200, e)
}

func DeleteEndpoint(ctx *Context) {
	pre := time.Now()
	id := ctx.ParamsInt64(":id")
	err := sqlstore.DeleteEndpoint(id, ctx.OrgId)
	if err != nil {
		log.Error(3, "api.DeleteEndpoint failed. %s", err)
		ctx.JSON(500, err)
		endpointDeleteFail.Inc(1)
		return
	}
	endpointDeleteDuration.Value(time.Now().Sub(pre))
	endpointDeleteOk.Inc(1)
	ctx.JSON(200, "ok")
}

func DiscoverEndpoint(ctx *Context, cmd model.DiscoverEndpointCmd) {
	checks, err := endpoint_discovery.Discover(cmd.Name)
	if err != nil {
		log.Error(3, "api.DiscoverEndpoint failed. %s", err)
		ctx.JSON(500, err)
	}
	ctx.JSON(200, checks)
}
