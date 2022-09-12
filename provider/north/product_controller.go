package north

import (
	"encoding/json"
	"go-iot/models"
	device "go-iot/models/device"

	"github.com/beego/beego/v2/server/web"
)

// 产品管理
func init() {
	web.Router("/product/list", &ProductController{}, "post:List")
	web.Router("/product/add", &ProductController{}, "post:Add")
	web.Router("/product/update", &ProductController{}, "post:Update")
	web.Router("/product/delete", &ProductController{}, "post:Delete")
}

type ProductController struct {
	web.Controller
}

// 查询设备列表
func (ctl *ProductController) List() {
	var ob models.PageQuery
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := device.ListProduct(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

// 添加设备
func (ctl *ProductController) Add() {
	data := ctl.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Url: ctl.Ctx.Input.URL(), Data: data}
	ctl.Data["json"] = ledSender.Add(request)
	ctl.ServeJSON()
}

// 更新设备信息
func (ctl *ProductController) Update() {
	data := ctl.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Url: ctl.Ctx.Input.URL(), Data: data}
	ctl.Data["json"] = ledSender.Update(request)
	ctl.ServeJSON()
}

// 删除设备
func (ctl *ProductController) Delete() {
	data := ctl.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: ctl.Ctx.Input.IP(), Url: ctl.Ctx.Input.URL(), Data: data}
	ctl.Data["json"] = ledSender.Delete(request)
	ctl.ServeJSON()
}
