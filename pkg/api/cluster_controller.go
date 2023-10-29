package api

import (
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core/common"
	"net/http"
)

// 集群管理
func init() {
	// 集群保活接口
	web.RegisterAPI("/cluster/keepalive", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
		if ctl.IsNotClusterRequest() {
			ctl.RespErr(common.NewErrCode(http.StatusMethodNotAllowed))
			return
		}
		var ob cluster.ClusterNode
		ctl.BindJSON(&ob)
		cluster.Keepalive(ob)
		ctl.RespOk()
	})
}
