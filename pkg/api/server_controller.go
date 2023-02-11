package api

import (
	"go-iot/pkg/codec"
	"go-iot/pkg/models"
	product "go-iot/pkg/models/device"
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
	var ob models.PageQuery[models.Network]
	err := ctl.BindJSON(&ob)
	if err != nil {
		ctl.RespError(err)
		return
	}

	res, err := network.ListNetwork(&ob)
	if err != nil {
		ctl.RespError(err)
	} else {
		ctl.RespOkData(res)
	}
}

func (ctl *ServerController) Add() {
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
	var resp models.JsonResp
	id := ctl.Param(":id")
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

func convertCodecNetwork(nw models.Network) (codec.NetworkConf, error) {
	pro, err := product.GetProductMust(nw.ProductId)
	if err != nil {
		return codec.NetworkConf{}, err
	}
	config := codec.NetworkConf{
		Name:          nw.Name,
		Port:          nw.Port,
		ProductId:     nw.ProductId,
		Configuration: nw.Configuration,
		Script:        pro.Script,
		Type:          nw.Type,
		CodecId:       pro.CodecId,
		CertBase64:    nw.CertBase64,
		KeyBase64:     nw.KeyBase64,
	}
	return config, nil
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
