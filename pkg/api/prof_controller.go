package api

import (
	// "runtime/pprof"

	"go-iot/pkg/api/web"
	"net/http"
	"net/http/pprof"
	"strings"
)

func init() {
	var profResource = Resource{
		Id:   "prof",
		Name: "系统运行状态",
		Action: []ResourceAction{
			QueryAction,
		},
	}
	RegResource(profResource)

	Get := func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(profResource, QueryAction) {
			return
		}
		ctl.Request.URL.Path = strings.Replace(ctl.Request.URL.Path, "/api/prof", "/debug/pprof", 1)
		pprof.Index(ctl.ResponseWriter, ctl.Request)
	}
	web.RegisterAPI("/prof", "GET", Get)
	web.RegisterAPI("/prof/allocs", "GET", Get)
	web.RegisterAPI("/prof/block", "GET", Get)
	web.RegisterAPI("/prof/goroutine", "GET", Get)
	web.RegisterAPI("/prof/cmdline", "GET", Get)
	web.RegisterAPI("/prof/heap", "GET", Get)
	web.RegisterAPI("/prof/mutex", "GET", Get)
	web.RegisterAPI("/prof/profile", "GET", Get)
	web.RegisterAPI("/prof/threadcreate", "GET", Get)
	web.RegisterAPI("/prof/trace", "GET", Get)

}
