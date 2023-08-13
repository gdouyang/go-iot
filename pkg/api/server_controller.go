package api

import (
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	"go-iot/pkg/models/network"
	"go-iot/pkg/network/servers"
	"strconv"
)

// 服务端管理
func init() {
	web.RegisterAPI("/server/list", "POST", &ServerController{}, "List")
	web.RegisterAPI("/server", "POST", &ServerController{}, "Add")
	web.RegisterAPI("/server", "PUT", &ServerController{}, "Update")
	web.RegisterAPI("/server/{id}", "DELETE", &ServerController{}, "Delete")
	web.RegisterAPI("/server/start/{id}", "POST", &ServerController{}, "Start")
	web.RegisterAPI("/server/meters/{id}", "GET", &ServerController{}, "Meters")
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
	id := ctl.Param("id")
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
	id := ctl.Param("id")
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
	m := map[string]interface{}{}
	if s != nil {
		m["totalConnection"] = s.TotalConnection()
	}
	ctl.RespOkData(m)
}
