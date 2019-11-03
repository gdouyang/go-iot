package xixun

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	models "go-iot/models"
	operates "go-iot/models/operates"
	"io/ioutil"
	"os"
	"strings"

	"github.com/astaxie/beego"
)

// 厂商ID
var (
	providerId           string           = "xixunled"
	ProviderImplXiXunLed ProviderXiXunLed = ProviderXiXunLed{providerId}
)

func init() {
	//启动WebSocket
	startWebSocket()
	// 注册厂商
	operates.RegisterProvider(ProviderImplXiXunLed.ProviderId(), ProviderImplXiXunLed)
}

// 厂商实现
type ProviderXiXunLed struct {
	Id string //厂商ID
}

func (this ProviderXiXunLed) ProviderId() string {
	return this.Id
}

// 开关操作
func (this ProviderXiXunLed) Switch(status []models.SwitchStatus, device operates.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "callCardService","fn": "setScreenOpen","arg1": %s}`
	if len(status) > 0 {
		ss := status[0]
		if ss.Status == "open" {
			abc = fmt.Sprintf(abc, "true")
		} else {
			abc = fmt.Sprintf(abc, "false")
		}
		resp, err := SendCommand(device.Sn, abc)
		if err != nil {
			rsp.Msg = err.Error()
		} else {
			rsp.Msg = resp
			rsp.Success = true
		}

	}
	return rsp
}

// Led 调光
func (this ProviderXiXunLed) Light(value int, device operates.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "callCardService","fn": "setBrightness","arg1": %d}`
	abc = fmt.Sprintf(abc, value)
	resp, err := SendCommand(device.Sn, abc)
	if err != nil {
		rsp.Msg = err.Error()
	} else {
		rsp.Msg = resp
		rsp.Success = true
	}
	return rsp
}

// Led 音量
func (this ProviderXiXunLed) Volume(value int, sn string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "callCardService","fn": "setVolume","arg1": %d}`
	abc = fmt.Sprintf(abc, value)
	resp, err := SendCommand(sn, abc)
	if err != nil {
		rsp.Msg = err.Error()
	} else {
		rsp.Msg = resp
		rsp.Success = true
	}
	return rsp
}

// 文件上传 url为文件下载路径，path为文件存储在本地路径  "/abc/portoflove.zip"
func (this ProviderXiXunLed) FileUpload(sn string, url string, filename string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "downloadFileToLocal","url": "%s","path": "/abc/%s"}`
	abc = fmt.Sprintf(abc, url, filename)
	resp, err := SendCommand(sn, abc)
	if err != nil {
		rsp.Msg = err.Error()
	} else {
		rsp.Msg = resp
		rsp.Success = true
	}
	return rsp
}

// 查询文件长度，用来判断文件是否完整，建议播放之前查看，或者上传查看
// Return:{"length":2560812,"_type":"success"}
type uploadResp struct {
	Type   string `json:"_type"`
	Length int64  `json:"length"`
}

func (this ProviderXiXunLed) FileLength(filename string, sn string) (int64, error) {
	abc := `{"type": "getLocalFileLength","path": "/abc/%s"}`
	abc = fmt.Sprintf(abc, filename)
	resp, err := SendCommand(sn, abc)
	if err != nil {
		return 0, err
	}
	rsp := uploadResp{}
	json.Unmarshal([]byte(resp), &rsp)
	if strings.EqualFold(rsp.Type, "success") {
		return rsp.Length, nil
	}
	return 0, errors.New(resp)
}

// 文件删除
func (this ProviderXiXunLed) FileDrop(filename string, sn string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "deleteFileFromLocal","path": "/abc/%s"}`
	abc = fmt.Sprintf(abc, filename)
	resp, err := SendCommand(sn, abc)
	if err != nil {
		rsp.Msg = err.Error()
	} else {
		rsp.Msg = resp
		rsp.Success = true
	}
	return rsp
}

// 文件播放ZIP
func (this ProviderXiXunLed) PlayZip(filename string, sn string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type":"commandXixunPlayer","command":{"_type":"PlayXixunProgramZip","path":"\/data\/data\/com.xixun.xy.conn\/files\/local\/abc\/%s","password":"888"}}`
	abc = fmt.Sprintf(abc, filename)
	resp, err := SendCommand(sn, abc)
	if err != nil {
		rsp.Msg = err.Error()
	} else {
		rsp.Msg = resp
		rsp.Success = true
	}
	return rsp
}

// 获取截图
type screenshoot struct {
	Type   string `json:"_type"`
	Result string `json:"result"`
}

func (this ProviderXiXunLed) ScreenShot(sn string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type":"callCardService","fn":"screenshot","arg1": 100,"arg2": 100}`
	resp, err := SendCommand(sn, abc)
	if err != nil {
		rsp.Msg = err.Error()
		rsp.Success = false
		return rsp
	}
	//截图保存在文件中，让界面默认展示
	ssback := screenshoot{}
	err = json.Unmarshal([]byte(resp), &ssback)
	if err != nil {
		rsp.Msg = err.Error()
		rsp.Success = false
		return rsp
	}
	if len(ssback.Result) == 0 && len(ssback.Type) == 0 {
		rsp.Msg = resp
		rsp.Success = false
		return rsp
	}
	pngStream, _ := base64.StdEncoding.DecodeString(ssback.Result)
	err = os.Mkdir("files/screenshot", 0700)
	if err != nil {
		beego.Info(err)
	}
	pngName := fmt.Sprintf("files/screenshot/%s.png", sn)
	err = ioutil.WriteFile(pngName, pngStream, 0666)
	if err != nil {
		rsp.Msg = err.Error()
		rsp.Success = false
		return rsp
	}
	beego.Info("截图指令:", resp)
	rsp.Msg = ssback.Result
	rsp.Success = true
	return rsp
}

//下发滚动文字
type MsgParam struct {
	Type      string `json:"type"`
	Method    string `json:"method"`
	Num       int    `json:"num"`
	Html      string `json:"html"`
	Interval  int    `json:"interval"`
	Direction string `json:"direction"`
	Align     string `json:"align"`
}

func (this ProviderXiXunLed) MsgPublish(sn string, msg MsgParam) operates.OperResp {
	var rsp operates.OperResp
	msg.Type = "invokeBuildInJs"
	msg.Method = "scrollMarquee"
	abc, _ := json.Marshal(msg)
	resp, err := SendCommand(sn, string(abc))
	if err != nil {
		rsp.Msg = err.Error()
		rsp.Success = false
		return rsp
	}
	rsp.Msg = resp
	rsp.Success = true
	return rsp
}

//清除顶层
func (this ProviderXiXunLed) Clear(sn string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type":"clear"}`
	resp, err := SendCommand(sn, abc)
	if err != nil {
		rsp.Msg = err.Error()
		rsp.Success = false
		return rsp
	}
	rsp.Msg = resp
	rsp.Success = true
	return rsp
}
