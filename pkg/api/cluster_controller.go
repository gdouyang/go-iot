package api

import (
	"errors"
	"go-iot/pkg/core/cluster"

	"github.com/beego/beego/v2/server/web"
)

// 集群管理
func init() {
	ns := web.NewNamespace("/api/cluster",
		web.NSRouter("/keepalive", &ClusterController{}, "post:Keepalive"),
	)
	web.AddNamespace(ns)
}

type ClusterController struct {
	RespController
}

func (ctl *ClusterController) Keepalive() {
	if ctl.isNotClusterRequest() {
		ctl.Ctx.Output.Status = 404
		ctl.RespError(errors.New("NotFound"))
		return
	}
	var ob cluster.ClusterNode
	ctl.BindJSON(&ob)
	cluster.Keepalive(ob)
	ctl.RespOk()
}
