// 定义操作接口

package operates

import (
	"errors"
	"fmt"
	"go-iot/models"

	"github.com/astaxie/beego"
)

func init() {
	// 监听设备在线状态，并修改数据库中的状态
	go func() {
		// 处理在线状态事件
		for {
			o := <-onlineChannel
			beego.Info("UpdateOnlineStatus")
			models.UpdateOnlineStatus(o.OnlineStatus, o.Sn, o.Provider)
		}
	}()
}

// 厂商map
var (
	providerMap   = map[string]IProvider{}
	onlineChannel = make(chan DeviceOnlineStatus, 10)
)

// 注册厂商
func RegisterProvider(id string, provider IProvider) {
	_, ok := providerMap[id]
	if ok {
		panic(fmt.Sprintf("provider id %s is exist", id))
	}
	providerMap[id] = provider
}

// 获取厂商实现
func GetProvider(id string) (interface{}, error) {
	provider, ok := providerMap[id]
	if ok {
		return provider, nil
	}
	return nil, errors.New("没有找到厂商")
}

// 厂商接口
type IProvider interface {
	//厂商ID全局唯一
	ProviderId() string
}

// 设备在线
type DeviceOnlineStatus struct {
	Sn           string
	Provider     string
	OnlineStatus string // onLine offLine
}

// 发布在线状态事件
func FireOnlineStatus(o DeviceOnlineStatus) {
	onlineChannel <- o
}

type OperResp struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

// 开头操作
type ISwitchOper interface {
	// 开关
	Switch(status []models.SwitchStatus, device models.Device) OperResp
}

// 调光操作
type ILightOper interface {
	// 调光
	// value 0-100
	Light(value int, device models.Device) OperResp
}
