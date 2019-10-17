package controllers

import (
	"go-iot/models/material"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/file/?:id", &FileRootController{}, "get:File")
}

type FileRootController struct {
	beego.Controller
}

// 下载素材
func (this *FileRootController) File() {
	id := this.Ctx.Input.Param(":id")

	ob, err := material.GetMaterialById(id)
	if err != nil {
		beego.Error(err.Error())
	}

	if len(ob.Id) == 0 {
		this.Ctx.Output.SetStatus(404)
	} else {
		this.Ctx.Output.Download("." + ob.Path)
	}

}
