package api

import (
	"go-iot/pkg/api/web"
	"go-iot/pkg/models"
	"go-iot/pkg/models/network"
	"go-iot/pkg/network/servers"
	"net/http"
	"strconv"
)

// 服务端管理
func init() {
	web.RegisterAPI("/server/list", "POST", func(w http.ResponseWriter, r *http.Request) {
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
	web.RegisterAPI("/server", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	web.RegisterAPI("/server", "PUT", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(sysConfigResource, SaveAction) {
			return
		}
		var ob models.Network
		err := ctl.BindJSON(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		err = network.UpdateNetwork(&ob)
		if err != nil {
			ctl.RespError(err)
			return
		}
		ctl.RespOk()
	})
	web.RegisterAPI("/server/{id}", "DELETE", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
	web.RegisterAPI("/server/start/{id}", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
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
	})
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
