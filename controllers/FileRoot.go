package controllers

import (
	"go-iot/provider/utils"
	"net/http"

	"github.com/astaxie/beego"
)

func init() {
	beego.Router("/file/?:name", &FileRootController{}, "get:File")
}

type FileRootController struct {
	beego.Controller
}

// 下载素材
func (this *FileRootController) File() {
	name := this.Ctx.Input.Param(":name")

	path := "./files/" + name
	exists, _ := utils.FileExists(path)
	if !exists {
		http.Error(this.Ctx.ResponseWriter, "file not found", 404)
	} else {
		this.Ctx.Output.Download(path)
	}
}
