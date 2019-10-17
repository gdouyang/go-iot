package xixun

import (
	"errors"
	"fmt"
	models "go-iot/models"
	operates "go-iot/models/operates"
	"net"

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
func (this ProviderXiXunLed) Switch(status []models.SwitchStatus, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "callCardService","fn": "setScreenOpen","arg1": %s}`
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
	abc := `{"type": "callCardService","fn": "setBrightness","arg1": %d}`
	abc = fmt.Sprintf(abc, value)
	resp := SendCommand(device.Sn, abc)
	beego.Info("light resp:", resp)
	rsp.Msg = resp
	return rsp
}

// Led 音量
func (this ProviderXiXunLed) Volume(value int, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "callCardService","fn": "setVolume","arg1": %d}`
	abc = fmt.Sprintf(abc, value)
	resp := SendCommand(device.Sn, abc)
	beego.Info("set volume resp:", resp)
	rsp.Msg = resp
	return rsp
}

// 文件上传 url为文件下载路径，path为文件存储在本地路径  "/abc/portoflove.zip"
func (this ProviderXiXunLed) FileUpload(sn string, url string, filename string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "downloadFileToLocal","url": "%s","path": "/abc/%s"}`
	abc = fmt.Sprintf(abc, url, filename)
	resp := SendCommand(sn, abc)
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

func (this ProviderXiXunLed) FileLength(filename string, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "getFileLength","path": "%s"}`
	abc = fmt.Sprintf(abc, filename)
	resp := SendCommand(device.Sn, abc)
	beego.Info("fileLength resp:", resp)
	rsp.Msg = resp
	return rsp
}

// 文件删除
func (this ProviderXiXunLed) FileDrop(filename string, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "deleteFileFromLocal","path": "/abc/%s"}`
	abc = fmt.Sprintf(abc, filename)
	resp := SendCommand(device.Sn, abc)
	beego.Info("fileLength resp:", resp)
	rsp.Msg = resp
	return rsp
}

// 文件播放ZIP
func (this ProviderXiXunLed) PlayZip(filename string, device models.Device) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type":"commandXixunPlayer","command":{"_type":"PlayXixunProgramZip","path":"/abc/%s","password":"888"}}`
	abc = fmt.Sprintf(abc, filename)
	resp := SendCommand(device.Sn, abc)
	beego.Info("fileLength resp:", resp)
	rsp.Msg = resp
	return rsp
}

// 获取截图
func (this ProviderXiXunLed) ScreenShot(sn string) operates.OperResp {
	var rsp operates.OperResp
	abc := `{"type": "callCardService","fn": "screenshot","arg1": 100,arg2": 100}`
	resp := SendCommand(sn, abc)
	beego.Info("fileLength resp:", resp)
	rsp.Msg = resp
	return rsp
}

// 下发实时消息

func externalIP() (net.IP, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			ip := getIpFromAddr(addr)
			if ip == nil {
				continue
			}
			return ip, nil
		}
	}
	return nil, errors.New("connected to the network?")
}

func getIpFromAddr(addr net.Addr) net.IP {
	var ip net.IP
	switch v := addr.(type) {
	case *net.IPNet:
		ip = v.IP
	case *net.IPAddr:
		ip = v.IP
	}
	if ip == nil || ip.IsLoopback() {
		return nil
	}
	ip = ip.To4()
	if ip == nil {
		return nil // not an ipv4 address
	}

	return ip
}
