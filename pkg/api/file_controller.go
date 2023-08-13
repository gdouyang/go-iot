package api

import (
	"encoding/base64"
	"go-iot/pkg/api/web"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func init() {
	web.RegisterAPI("/file/base64", "POST", &FileRootController{}, "Base64")
	web.RegisterAPI("/file/upload", "POST", &FileRootController{}, "Upload")
}

type FileRootController struct {
	AuthController
}

func (ctl *FileRootController) Base64() {
	f, _, err := ctl.Request.FormFile("file")
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
	f, h, err := ctl.Request.FormFile("file")
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

	dst, err := os.OpenFile("./files/"+fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o666)
	if err != nil {
		ctl.RespError(err)
		return
	}
	defer dst.Close()

	_, err = io.CopyBuffer(dst, f, make([]byte, 1024*32))
	if err != nil {
		ctl.RespError(err)
		return
	}
	ctl.RespOkData("api/file/" + fileName)
}
