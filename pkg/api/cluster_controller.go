package api

import (
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/common"
	"go-iot/pkg/core"
	"net/http"
)

// 集群管理
func init() {
	// 集群保活：请求体为对端节点；响应体为本节点（回填 Name/Index）
	web.RegisterAPI("/cluster/keepalive", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
		if ctl.IsNotClusterRequest() {
			ctl.RespErr(common.NewErrCode(http.StatusMethodNotAllowed))
			return
		}
		var ob cluster.ClusterNode
		ctl.BindJSON(&ob)
		cluster.Keepalive(ob)
		ctl.Resp(common.JsonRespOkData(cluster.LocalNode()))
	})

	// 节点视图：集群 token 或登录用户（管理端需 sys-config 查询权限）
	web.RegisterAPI("/cluster/nodes", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
		if ctl.IsNotClusterRequest() {
			auth := NewAuthController(w, r)
			if auth.isForbidden(sysConfigResource, QueryAction) {
				return
			}
			auth.RespOkData(cluster.ListNodes())
			return
		}
		ctl.RespOkData(cluster.ListNodes())
	})

	// 集群健康：语义类似 ES _cluster/health（green/yellow/red），本节点 Registry 视角
	web.RegisterAPI("/cluster/health", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
		if ctl.IsNotClusterRequest() {
			auth := NewAuthController(w, r)
			if auth.isForbidden(sysConfigResource, QueryAction) {
				return
			}
			auth.RespOkData(cluster.Health())
			return
		}
		ctl.RespOkData(cluster.Health())
	})

	// 集群内部功能调用（请求-响应）。ForceLocal，避免路由环。
	web.RegisterAPI("/cluster/cmd-invoke", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
		if ctl.IsNotClusterRequest() {
			ctl.RespErr(common.NewErrCode(http.StatusMethodNotAllowed))
			return
		}
		var msg core.FuncInvoke
		if err := ctl.BindJSON(&msg); err != nil {
			ctl.RespError(err)
			return
		}
		err := core.GetDeviceGateway().Invoke(r.Context(), msg, core.InvokeOptions{
			OfflineCache: msg.OfflineCache,
			ForceLocal:   true,
		})
		if err != nil {
			if err.Code == http.StatusOK {
				ctl.Resp(common.JsonResp{Success: true, Msg: err.Message, Code: err.Code})
				return
			}
			ctl.RespErr(err)
			return
		}
		ctl.RespOk()
	})
}
