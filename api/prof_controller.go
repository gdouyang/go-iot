package api

import (
	// "runtime/pprof"

	"net/http/pprof"
	"strings"

	"github.com/beego/beego/v2/server/web"
)

var profResource = Resource{
	Id:   "prof",
	Name: "系统运行状态",
	Action: []ResourceAction{
		QueryAction,
	},
}

func init() {
	ns := web.NewNamespace("/api/prof",
		web.NSRouter("/", &ProfController{}, "get:Get"),
		web.NSRouter("/allocs", &ProfController{}, "get:Get"),
		web.NSRouter("/block", &ProfController{}, "get:Get"),
		web.NSRouter("/goroutine", &ProfController{}, "get:Get"),
		web.NSRouter("/cmdline", &ProfController{}, "get:Get"),
		web.NSRouter("/heap", &ProfController{}, "get:Get"),
		web.NSRouter("/mutex", &ProfController{}, "get:Get"),
		web.NSRouter("/profile", &ProfController{}, "get:Get"),
		web.NSRouter("/threadcreate", &ProfController{}, "get:Get"),
		web.NSRouter("/trace", &ProfController{}, "get:Get"),
	)
	web.AddNamespace(ns)

	regResource(profResource)
}

type ProfController struct {
	AuthController
}

func (ctl *ProfController) Get() {
	if ctl.isForbidden(profResource, QueryAction) {
		return
	}
	defer func() {}()
	ctl.Ctx.Request.URL.Path = strings.Replace(ctl.Ctx.Request.URL.Path, "/api/prof/", "/debug/pprof/", 1)
	pprof.Index(ctl.Ctx.ResponseWriter.ResponseWriter, ctl.Ctx.Request)
}
