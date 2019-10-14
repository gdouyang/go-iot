package xixun

import (
	"fmt"
	models "go-iot/models"
	operates "go-iot/models/operates"

	"github.com/astaxie/beego"
)

// 厂商ID
var providerId string = "xixunled"

func init() {
	//启动WebSocket
	startWebSocket()
	// 注册厂商
	provider := ProviderXiXunLed{providerId}
	operates.RegisterProvider(provider.ProviderId(), provider)
}

// 厂商实现
type ProviderXiXunLed struct {
	Id string //厂商ID
}

func (this ProviderXiXunLed) ProviderId() string {
	return this.Id
}

// 开关操作
func (this ProviderXiXunLed) Switch(status []models.SwitchStatus, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := "{\"type\": \"callCardService\",\"fn\": \"setScreenOpen\",\"arg1\": %s}"
	if len(status) > 0 {
		ss := status[0]
		if ss.Status == "open" {
			abc = fmt.Sprintf(abc, "true")
		} else {
			abc = fmt.Sprintf(abc, "false")
		}
		resp := SendCommand(device.Sn, abc)
		beego.Info("switch resp:", resp)
		rsp.Msg = resp
	}
	return rsp
}

// Led 调光
func (this ProviderXiXunLed) Light(value int, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := "{\"type\": \"callCardService\",\"fn\": \"setBrightness\",\"arg1\": %d}"
	abc = fmt.Sprintf(abc, value)
	resp := SendCommand(device.Sn, abc)
	beego.Info("light resp:", resp)
	rsp.Msg = resp
	return rsp
}

// Led 音量
func (this ProviderXiXunLed) Volume(value int, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := "{\"type\": \"callCardService\",\"fn\": \"setVolume\",\"arg1\": %d}"
	abc = fmt.Sprintf(abc, value)
	resp := SendCommand(device.Sn, abc)
	beego.Info("set volume resp:", resp)
	rsp.Msg = resp
	return rsp
}

// 文件上传 url为文件下载路径，path为文件存储在本地路径  "/abc/portoflove.zip"
func (this ProviderXiXunLed) FileUpload(url string, path string, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := "{\"type\": \"downloadFileToLocal\",\"url\": \"%s\",\"path\": \"%s\"}"
	abc = fmt.Sprintf(abc, url, path)
	resp := SendCommand(device.Sn, abc)
	beego.Info("Upload file resp:", resp)
	rsp.Msg = resp
	return rsp
}

// 查询文件长度，用来判断文件是否完整，建议播放之前查看，或者上传查看
// Return:{"length":2560812,"_type":"success"}
type uploadResp struct {
	Type   string `json:"_type"`
	Length int    `json:"length"`
}

func (this ProviderXiXunLed) FileLength(path string, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := "{\"type\": \"getFileLength\",\"path\": \"%s\"}"
	abc = fmt.Sprintf(abc, path)
	resp := SendCommand(device.Sn, abc)
	beego.Info("fileLength resp:", resp)
	rsp.Msg = resp
	return rsp
}
