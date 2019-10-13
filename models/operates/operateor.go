// 定义操作接口

package operates

import (
	"fmt"
	"go-iot/models"
)

// 厂商map
var providerMap = map[string]IProvider{}

// 注册厂商
func RegisterProvider(id string, provider IProvider) {
	_, ok := providerMap[id]
	if ok {
		panic(fmt.Sprintf("provider id %s is exist", id))
	}
	providerMap[id] = provider
}

// 获取厂商实现
func GetProvider(id string) interface{} {
	return providerMap[id]
}

// 厂商接口
type IProvider interface {
	//厂商ID全局唯一
	ProviderId() string
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
