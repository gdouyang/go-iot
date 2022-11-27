package api

import (
	"encoding/base64"
	"go-iot/models"
	"go-iot/network/util"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beego/beego/v2/server/web"
)

func init() {
	web.Router("/api/file/?:name", &FileAnonController{}, "get:File")
	web.Router("/api/file/base64", &FileRootController{}, "post:Base64")
	web.Router("/api/file/upload", &FileRootController{}, "post:Upload")
}

type FileAnonController struct {
	web.Controller
}

func (ctl *FileAnonController) File() {
	name := ctl.Ctx.Input.Param(":name")

	path := "./files/" + name
	exists, _ := util.FileExists(path)
	if !exists {
		http.Error(ctl.Ctx.ResponseWriter, "file not found", 404)
	} else {
		ctl.Ctx.Output.Download(path)
	}
}

type FileRootController struct {
	AuthController
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

func (ctl *FileRootController) Upload() {
	var resp = models.JsonRespOk()
	defer func() {
		ctl.Data["json"] = resp
		ctl.ServeJSON()
	}()
	f, h, err := ctl.GetFile("file")
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	defer f.Close()
	fileName := h.Filename
	index := strings.LastIndex(fileName, ".")
	if index != -1 {
		fileName = fileName[:index] + strconv.Itoa(int(time.Now().Unix())) + fileName[index:]
	}
	os.Mkdir("./files", os.ModePerm)
	err = ctl.SaveToFile("file", "./files/"+fileName)
	if err != nil {
		resp = models.JsonRespError(err)
		return
	}
	resp.Data = "api/file/" + fileName
}
