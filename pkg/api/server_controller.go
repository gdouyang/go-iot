package api

import (
	"errors"
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	"go-iot/pkg/models/network"
	"go-iot/pkg/network/servers"
	"net/http"
	"strconv"
)

// 服务端管理
func init() {
	var netConfigResource = Resource{
		Id:   "network-config",
		Name: "网络管理",
		Action: []ResourceAction{
			QueryAction,
			CretaeAction,
			SaveAction,
			DeleteAction,
		},
	}
	RegResource(netConfigResource)
	// 分页查询
	web.RegisterAPI("/server/page", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	// 新增
	web.RegisterAPI("/server", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(netConfigResource, CretaeAction) {
			return
		}
		var ob models.Network
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		nw, err := network.GetNetworkByPort(ob.Port)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if nw != nil {
			ctl.RespError(errors.New("端口已被使用，请更换"))
			return
		}
		err = network.AddNetWork(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 修改
	web.RegisterAPI("/server", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(netConfigResource, SaveAction) {
			return
		}
		var ob models.Network
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		nw, err := network.GetNetworkByPort(ob.Port)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if nw != nil && nw.Id != ob.Id {
			ctl.RespError(errors.New("端口已被使用，请更换"))
			return
		}
		err = network.UpdateNetwork(&models.Network{Id: ob.Id, Port: ob.Port})
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 单个查询
	web.RegisterAPI("/server/{id}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(netConfigResource, QueryAction) {
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
		ctl.RespOkData(nw)
	})
	// 删除
	web.RegisterAPI("/server/{id}", "DELETE", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(netConfigResource, DeleteAction) {
			return
		}
		var ob models.Network
		id := ctl.Param("id")
		_id, err := strconv.Atoi(id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ob.Id = int64(_id)
		nw, err := network.GetNetwork(ob.Id)
		if err != nil {
			ctl.RespError(err)
			return
		}
		if len(nw.ProductId) > 0 {
			ctl.RespError(errors.New("网络已管理产品无法删除"))
			return
		}
		err = network.DeleteNetwork(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	// 启动
	web.RegisterAPI("/server/start/{id}", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(netConfigResource, SaveAction) {
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
	})
	// 查看连接信息
	web.RegisterAPI("/server/meters/{id}", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
}
