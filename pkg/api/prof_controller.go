package api

import (
	// "runtime/pprof"

	"go-iot/pkg/api/web"
	"net/http/pprof"
	"strings"
)

var profResource = Resource{
	Id:   "prof",
	Name: "系统运行状态",
	Action: []ResourceAction{
		QueryAction,
	},
}

func init() {
	web.RegisterAPI("/prof", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/allocs", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/block", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/goroutine", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/cmdline", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/heap", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/mutex", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/profile", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/threadcreate", "GET", &ProfController{}, "Get")
	web.RegisterAPI("/prof/trace", "GET", &ProfController{}, "Get")

	RegResource(profResource)
}

type ProfController struct {
	AuthController
}

func (ctl *ProfController) Get() {
	if ctl.isForbidden(profResource, QueryAction) {
		return
	}
	ctl.Request.URL.Path = strings.Replace(ctl.Request.URL.Path, "/api/prof", "/debug/pprof", 1)
	pprof.Index(ctl.ResponseWriter, ctl.Request)
}
