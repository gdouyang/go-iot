package api

import (
	"encoding/base64"
	"go-iot/pkg/api/web"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// 文件上传/转换：挂在 file-mgr 资源下，需登录且具备对应动作。
var fileResource = Resource{
	Id:   "file-mgr",
	Name: "文件",
	Sort: 200, // 非侧栏主菜单，排在业务菜单之后
	Action: []ResourceAction{
		QueryAction,
		SaveAction,
	},
}

func init() {
	RegResource(fileResource)
	// 文件转base64
	web.RegisterAPI("/file/base64", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(fileResource, SaveAction) {
			return
		}
		f, _, err := ctl.FormFile("file")
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
	})
	// 文件上传
	web.RegisterAPI("/file/upload", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(fileResource, SaveAction) {
			return
		}
		f, h, err := ctl.FormFile("file")
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

	})
}
