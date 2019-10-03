package controllers

import (
	"encoding/json"
	"go-iot/models"

	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2"
)

func init() {
	beego.Router("/device/list", &DeviceController{}, "post:List")
	beego.Router("/device/add", &DeviceController{}, "post:Add")
}

type DeviceController struct {
	beego.Controller
}

func (this *DeviceController) List() {
	mongoExecute("device", func(col *mgo.Collection) {
		var result []models.Device
		col.Find(nil).All(&result)

		this.Data["json"] = &models.PageResult{1, 1, 0, result}
		this.ServeJSON()
	})
}

func (this *DeviceController) Add() {
	var ob models.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	mongoExecute("device", func(col *mgo.Collection) {
		err := col.Insert(&ob)
		if err != nil {
			beego.Error("insert fail")
		}
		this.Data["json"] = &ob
		this.ServeJSON()
	})
}
