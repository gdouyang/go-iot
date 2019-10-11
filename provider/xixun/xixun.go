package xixun

import (
	"fmt"
	"go-iot/models"

	"github.com/astaxie/beego"
)

// 厂商ID
var providerId string = "xixunled"

func init() {
	//启动WebSocket
	startWebSocket()
	// 注册厂商
	provider := ProviderXiXunLed{providerId}
	models.RegisterProvider(provider.ProviderId(), provider)
}

// 厂商实现
type ProviderXiXunLed struct {
	Id string //厂商ID
}

func (this ProviderXiXunLed) ProviderId() string {
	return this.Id
}

// 开关操作
func (this ProviderXiXunLed) Switch(status []models.SwitchStatus, device models.Device) {
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
	}
}
