package api

import (
	"encoding/base64"
	"go-iot/codec/util"
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
	RespController
}

func (ctl *FileAnonController) File() {
	name := ctl.Param(":name")

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
	f, _, err := ctl.GetFile("file")
	if err != nil {
		if err.Error() != "http: no such file" {
			ctl.RespError(err)
			return
		}
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	if err != nil {
		ctl.RespError(err)
		return
	}
	base64Str := base64.StdEncoding.EncodeToString(b)
	ctl.RespOkData(base64Str)
}

func (ctl *FileRootController) Upload() {
	f, h, err := ctl.GetFile("file")
	if err != nil {
		ctl.RespError(err)
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
		ctl.RespError(err)
		return
	}
	ctl.RespOkData("api/file/" + fileName)
}
