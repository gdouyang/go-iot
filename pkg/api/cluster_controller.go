package api

import (
	"go-iot/pkg/api/web"
	"go-iot/pkg/cluster"
	"go-iot/pkg/core/common"
	"net/http"
)

// 集群管理
func init() {
	web.RegisterAPI("/cluster/keepalive", "POST", &ClusterController{}, "Keepalive")
}

type ClusterController struct {
	web.RespController
}

func (ctl *ClusterController) Keepalive() {
	if ctl.IsNotClusterRequest() {
		ctl.RespErr(common.NewErrCode(http.StatusMethodNotAllowed))
		return
	}
	var ob cluster.ClusterNode
	ctl.BindJSON(&ob)
	cluster.Keepalive(ob)
	ctl.RespOk()
}
