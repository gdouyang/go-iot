package api

import (
	"go-iot/codec"
	"go-iot/models"
	"go-iot/models/network"
	"go-iot/network/servers"
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
	ctl.BindJSON(&ob)

	res, err := network.ListNetwork(&ob)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *ServerController) Add() {
	var ob models.Network
	ctl.BindJSON(&ob)
	err := network.AddNetWork(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ServerController) Update() {
	var ob models.Network
	ctl.BindJSON(&ob)
	err := network.AddNetWork(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ServerController) Delete() {
	var ob models.Network
	ctl.BindJSON(&ob)
	err := network.DeleteNetwork(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func (ctl *ServerController) Start() {
	var resp models.JsonResp
	id := ctl.Ctx.Input.Param(":id")
	_id, err := strconv.Atoi(id)
	if err != nil {
		ctl.RespError(err)
		return
	}
	nw, err := network.GetNetwork(int64(_id))
	resp.Success = true
	if err != nil {
		ctl.RespError(err)
		return
	}
	config := convertCodecNetwork(nw)
	err = servers.StartServer(config)
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOk()
}

func convertCodecNetwork(nw models.Network) codec.NetworkConf {
	config := codec.NetworkConf{
		Name:          nw.Name,
		Port:          nw.Port,
		ProductId:     nw.ProductId,
		Configuration: nw.Configuration,
		Script:        nw.Script,
		Type:          nw.Type,
		CodecId:       nw.CodecId,
	}
	return config
}

func (ctl *ServerController) Meters() {
	id := ctl.Ctx.Input.Param(":id")
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
