package api

import (
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
	var ob cluster.ClusterNode
	ctl.BindJSON(&ob)
	cluster.Keepalive(ob)
	ctl.RespOk()
}
