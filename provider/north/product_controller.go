package north

import (
	"encoding/json"
	"go-iot/models"
	product "go-iot/models/device"

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

// 查询型号列表
func (ctl *ProductController) List() {
	var ob models.PageQuery
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)

	res, err := product.ListProduct(&ob)
	if err != nil {
		ctl.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		ctl.Data["json"] = &res
	}
	ctl.ServeJSON()
}

// 添加型号
func (ctl *ProductController) Add() {
	var ob models.Product
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = product.AddProduct(&ob)
	ctl.ServeJSON()
}

// 更新型号信息
func (ctl *ProductController) Update() {
	var ob models.Product
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = product.UpdateProduct(&ob)
	ctl.ServeJSON()
}

// 删除型号
func (ctl *ProductController) Delete() {
	var ob models.Product
	json.Unmarshal(ctl.Ctx.Input.RequestBody, &ob)
	ctl.Data["json"] = product.DeleteProduct(&ob)
	ctl.ServeJSON()
}
