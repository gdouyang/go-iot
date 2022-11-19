package api

import (
	"encoding/base64"
	"go-iot/models"
	"go-iot/network/util"
	"io"
	"net/http"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/file/?:name", &FileRootController{}, "get:File")
	web.Router("/api/file/base64", &FileRootController{}, "post:Base64")
}

type FileRootController struct {
	AuthController
}

// 下载素材
func (ctl *FileRootController) File() {
	name := ctl.Ctx.Input.Param(":name")

	path := "./files/" + name
	exists, _ := util.FileExists(path)
	if !exists {
		http.Error(ctl.Ctx.ResponseWriter, "file not found", 404)
	} else {
		ctl.Ctx.Output.Download(path)
	}
}

func (ctl *FileRootController) Base64() {
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	f, _, err := ctl.GetFile("file")
	if err != nil {
		if err.Error() != "http: no such file" {
			resp = models.JsonRespError(err)
			return
		}
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	base64Str := base64.StdEncoding.EncodeToString(b)
	resp.Data = base64Str
}
