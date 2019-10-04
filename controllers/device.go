package controllers

import (
	"encoding/json"
	"go-iot/models"

	"gopkg.in/mgo.v2/bson"

	"github.com/astaxie/beego"
	"gopkg.in/mgo.v2"
)

func init() {
	beego.Router("/device/list", &DeviceController{}, "post:List")
	beego.Router("/device/add", &DeviceController{}, "post:Add")
	beego.Router("/device/update", &DeviceController{}, "post:Update")
	beego.Router("/device/delete", &DeviceController{}, "post:Delete")
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

func (this *DeviceController) Update() {
	var ob models.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	mongoExecute("device", func(col *mgo.Collection) {
		err := col.Update(bson.M{"id": ob.Id}, ob)
		if err != nil {
			beego.Error("insert fail")
		}
		this.Data["json"] = &ob
		this.ServeJSON()
	})
}

func (this *DeviceController) Delete() {
	var ob models.Device
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	mongoExecute("device", func(col *mgo.Collection) {
		err := col.Remove(bson.M{"id": ob.Id})
		if err != nil {
			beego.Error("insert fail")
		}
		this.Data["json"] = &ob
		this.ServeJSON()
	})
}
