package controllers

import (
	"fmt"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/file/?:id", &FileRootController{}, "get:File")
}

type FileRootController struct {
	beego.Controller
}

func (this *FileRootController) File() {
	fname := this.Ctx.Input.Param(":id")
	this.Ctx.Output.Download(fmt.Sprintf("files/%s", fname), fname)
}
