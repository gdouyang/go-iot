package north

import (
	"encoding/json"
	"go-iot/models"
	device "go-iot/models/device"
	"go-iot/models/operates"

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
func (this *ProductController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := device.ListProduct(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加设备
func (this *ProductController) Add() {
	data := this.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = ledSender.Add(request)
	this.ServeJSON()
}

// 更新设备信息
func (this *ProductController) Update() {
	data := this.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = ledSender.Update(request)
	this.ServeJSON()
}

// 删除设备
func (this *ProductController) Delete() {
	data := this.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = ledSender.Delete(request)
	this.ServeJSON()
}

// 查询所有厂商
func (this *ProductController) ListProvider() {
	pros := operates.AllProvierId()
	this.Data["json"] = &pros
	this.ServeJSON()
}
