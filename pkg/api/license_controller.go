package api

import (
	"errors"
	"io"
	"net/http"

	"go-iot/pkg/api/web"
	"go-iot/pkg/license"
	logs "go-iot/pkg/logger"
)

var licenseResource = Resource{
	Id:   "license-mgr",
	Name: "授权管理",
	Sort: 110, // 系统管理下：授权管理
	Action: []ResourceAction{
		QueryAction,
		SaveAction,
	},
}

func init() {
	RegResource(licenseResource)

	// 获取当前系统 License 授权详情 (需鉴权)
	web.RegisterAPI("/system/license/info", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(licenseResource, QueryAction) {
			return
		}
		info := license.Default().GetInfo()
		ctl.RespOkData(info)
	})

	// 上传并激活新的 License 授权文件 (.lic) (需鉴权)
	web.RegisterAPI("/system/license/upload", "POST", func(w http.ResponseWriter, r *http.Request) {
		ctl := NewAuthController(w, r)
		if ctl.isForbidden(licenseResource, SaveAction) {
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			// 尝试从 Body 读取 JSON 字符串
			var bodyReq struct {
				Content string `json:"content"`
			}
			if bErr := ctl.BindJSON(&bodyReq); bErr == nil && len(bodyReq.Content) > 0 {
				if err := license.Default().SaveAndReload([]byte(bodyReq.Content)); err != nil {
					ctl.RespError(err)
					return
				}
				ctl.RespOkData(license.Default().GetInfo())
				return
			}
			ctl.RespError(errors.New("请上传 .lic 授权文件"))
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			ctl.RespError(err)
			return
		}

		if err := license.Default().SaveAndReload(data); err != nil {
			ctl.RespError(err)
			return
		}

		logs.Infof("license: uploaded by user %s", ctl.GetCurrentUser().Username)
		ctl.RespOkData(license.Default().GetInfo())
	})

	// 开放轻量级授权状态检测 (供前端路由拦截判断)
	web.RegisterAPI("/system/license/status", "GET", func(w http.ResponseWriter, r *http.Request) {
		ctl := web.NewController(w, r)
		info := license.Default().GetInfo()
		// 仅返回轻量关键状态
		res := map[string]interface{}{
			"status":             info.Status,
			"statusText":         info.StatusText,
			"isLicensed":         info.IsLicensed,
			"requireRedirect":    info.RequireRedirect,
			"machineFingerprint": info.MachineFingerprint,
			"customerName":       info.CustomerName,
			"expiresAtFormatted": info.ExpiresAtFormatted,
			"maxDevices":         info.MaxDevices,
			"currentDevices":     info.CurrentDevices,
		}
		ctl.RespOkData(res)
	})
}
