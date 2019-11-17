package north

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/led"
	"go-iot/models/operates"
	"go-iot/provider/north/sender"

	"github.com/astaxie/beego"
)

// 设备管理
func init() {
	beego.Router("/led/list", &LedController{}, "post:List")
	beego.Router("/north/led/add", &LedController{}, "post:Add")
	beego.Router("/north/led/update", &LedController{}, "post:Update")
	beego.Router("/north/led/delete", &LedController{}, "post:Delete")
	beego.Router("/led/listProvider", &LedController{}, "post:ListProvider")

}

var (
	ledSender sender.LedSender = sender.LedSender{CheckAgent: true}
)

type LedController struct {
	beego.Controller
}

// 查询设备列表
func (this *LedController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := led.ListDevice(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加设备
func (this *LedController) Add() {
	data := this.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = ledSender.Add(request)
	this.ServeJSON()
}

// 更新设备信息
func (this *LedController) Update() {
	data := this.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = ledSender.Update(request)
	this.ServeJSON()
}

// 删除设备
func (this *LedController) Delete() {
	data := this.Ctx.Input.RequestBody
	request := models.IotRequest{Ip: this.Ctx.Input.IP(), Url: this.Ctx.Input.URL(), Data: data}
	this.Data["json"] = ledSender.Delete(request)
	this.ServeJSON()
}

// 查询所有厂商
func (this *LedController) ListProvider() {
	pros := operates.AllProvierId()
	this.Data["json"] = &pros
	this.ServeJSON()
}
