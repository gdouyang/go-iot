package api

import (
	"go-iot/pkg/models"
	"go-iot/pkg/models/network"
	"go-iot/pkg/network/servers"
	"strconv"

	"github.com/beego/beego/v2/server/web"
)

// 服务端管理
func init() {
	ns := web.NewNamespace("/api/server",
		web.NSRouter("/list", &ServerController{}, "post:List"),
		web.NSRouter("/", &ServerController{}, "post:Add"),
		web.NSRouter("/", &ServerController{}, "put:Update"),
		web.NSRouter("/:id", &ServerController{}, "delete:Delete"),
		web.NSRouter("/start/:id", &ServerController{}, "get:Start"),
		web.NSRouter("/meters/:id", &ServerController{}, "get:Meters"),
	)
	web.AddNamespace(ns)
}

type ServerController struct {
	AuthController
}

func (ctl *ServerController) List() {
	var ob models.PageQuery
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := network.PageNetwork(&ob)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *ServerController) Add() {
	if ctl.isForbidden(sysConfigResource, SaveAction) {
		return
	}
	var ob models.Network
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = network.AddNetWork(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ServerController) Update() {
	if ctl.isForbidden(sysConfigResource, SaveAction) {
		return
	}
	var ob models.Network
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = network.AddNetWork(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ServerController) Delete() {
	if ctl.isForbidden(sysConfigResource, SaveAction) {
		return
	}
	var ob models.Network
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = network.DeleteNetwork(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ServerController) Start() {
	if ctl.isForbidden(sysConfigResource, SaveAction) {
		return
	}
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := network.GetNetwork(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	config, err := convertCodecNetwork(nw)
	if err != nil {
		ctl.RespError(err)
		return
	}
	err = servers.StartServer(config)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ServerController) Meters() {
	id := ctl.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := network.GetNetwork(int64(_id))
	if err != nil {
		ctl.RespError(err)
		return
	}
	s := servers.GetServer(nw.ProductId)
	if s != nil {
		m := map[string]interface{}{}
		m["totalConnection"] = s.TotalConnection()
		ctl.RespOkData(m)
	}
	ctl.RespOk()
}
