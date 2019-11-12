// 定义操作接口

package operates

import (
	"errors"
	"fmt"
	"go-iot/models"
)

type Device struct {
	Id       string `json:"id"` //设备ID
	Sn       string `json:"sn"` //设备SN
	Name     string `json:"name"`
	Provider string `json:"provider"` //厂商
	Agent    string `json:"agent"`    //
}

// 厂商map
var (
	providerMap = map[string]IProvider{}
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

// 返回所有厂商ID
func AllProvierId() []string {
	var providerNames []string
	for key := range providerMap {
		providerNames = append(providerNames, key)
	}
	return providerNames
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
	Type         string
}

type OperResp struct {
	Success bool   `json:"success"`
	Msg     string `json:"msg"`
}

// 开头操作
type ISwitchOper interface {
	// 开关
	Switch(status []models.SwitchStatus, device Device) OperResp
}

// 调光操作
type ILightOper interface {
	// 调光
	// value 0-100
	Light(value int, device Device) OperResp
}

//  在线状态
type IOnlineStatusOper interface {
	// 获取在线状态
	// 返回 onLine/offLine
	GetOnlineStatus(device Device) string
}
