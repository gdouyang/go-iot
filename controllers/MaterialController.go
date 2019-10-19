package controllers

import (
	"encoding/json"
	"go-iot/models"
	"go-iot/models/material"
	"strconv"
	"strings"
	"time"

	"github.com/astaxie/beego"
)

// 素材管理
func init() {
	beego.Router("/material/list", &MaterialController{}, "post:List")
	beego.Router("/material/add", &MaterialController{}, "post:Add")
	beego.Router("/material/update", &MaterialController{}, "post:Add")
	beego.Router("/material/delete", &MaterialController{}, "post:Delete")
}

type MaterialController struct {
	beego.Controller
}

// 查询素材列表
func (this *MaterialController) List() {
	var ob models.PageQuery
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)

	res, err := material.ListMaterial(&ob)
	if err != nil {
		this.Data["json"] = models.JsonResp{Success: false, Msg: err.Error()}
	} else {

		this.Data["json"] = &res
	}
	this.ServeJSON()
}

// 添加素材
func (this *MaterialController) Add() {
	var ob material.Material
	ob.Name = this.GetString("name")
	ob.Id = this.GetString("id")
	// ob.Type = this.GetString("type")
	var resp models.JsonResp
	resp.Success = true
	defer func() {
		this.Data["json"] = &resp
		this.ServeJSON()
	}()
	f, h, err := this.GetFile("uploadname")
	if err != nil {
		if err.Error() != "http: no such file" {
			resp.Msg = err.Error()
			return
		}
	} else {
		defer f.Close()
		fileName := h.Filename
		index := strings.LastIndex(fileName, ".")
		if index != -1 {
			fileName = fileName[:index] + strconv.Itoa(int(time.Now().Unix())) + fileName[index:]
		}
		filePath := "/files/" + fileName
		err = this.SaveToFile("uploadname", "."+filePath)
		if err != nil {
			resp.Msg = err.Error()
			return
		}
		ob.Path = filePath
		ob.Size = strconv.FormatInt(h.Size, 10)
	}
	if len(ob.Id) > 0 {
		resp.Msg = "修改成功!"
		// 保存入库
		err = material.UpdateMaterial(&ob)
	} else {
		resp.Msg = "添加成功!"
		// 保存入库
		err = material.AddMaterial(&ob)
	}
	if err != nil {
		resp.Msg = err.Error()
		resp.Success = false
	}
}

// 删除素材
func (this *MaterialController) Delete() {
	var ob material.Material
	json.Unmarshal(this.Ctx.Input.RequestBody, &ob)
	material.DeleteMaterial(&ob)
	this.Data["json"] = &ob
	this.ServeJSON()
}