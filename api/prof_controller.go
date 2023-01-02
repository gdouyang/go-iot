package api

import (
	"runtime/pprof"

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
	var resp map[string]interface{} = make(map[string]interface{})
	profiles := pprof.Profiles()
	for _, profile := range profiles {
		resp[profile.Name()] = profile.Count()
	}
	ctl.RespOkData(resp)
}
