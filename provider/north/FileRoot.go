package north

import (
	"go-iot/provider/util"
	"net/http"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/file/?:name", &FileRootController{}, "get:File")
}

type FileRootController struct {
	web.Controller
}

// 下载素材
func (this *FileRootController) File() {
	name := this.Ctx.Input.Param(":name")

	path := "./files/" + name
	exists, _ := util.FileExists(path)
	if !exists {
		http.Error(this.Ctx.ResponseWriter, "file not found", 404)
	} else {
		this.Ctx.Output.Download(path)
	}
}
