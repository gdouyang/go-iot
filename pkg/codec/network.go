package codec

import "sync"

// 服务器类型
type NetType string

const (
	// MQTT服务端
	MQTT_BROKER NetType = "MQTT_BROKER"
	// TCP服务端
	TCP_SERVER NetType = "TCP_SERVER"
	// HTTP服务端
	HTTP_SERVER NetType = "HTTP_SERVER"
	// WebSocket服务端
	WEBSOCKET_SERVER NetType = "WEBSOCKET_SERVER"

	// MQTT客户端
	MQTT_CLIENT NetType = "MQTT_CLIENT"
	// TCP客户端
	TCP_CLIENT NetType = "TCP_CLIENT"
	// MODBUS
	MODBUS NetType = "MODBUS"
)

func IsNetClientType(str string) bool {
	return TCP_CLIENT == NetType(str) || MQTT_CLIENT == NetType(str) || MODBUS == NetType(str)
}

// 网络配置
type NetworkConf struct {
	Name          string `json:"name"`
	Port          int32  `json:"port"`
	ProductId     string `json:"productId"`
	Configuration string `json:"configuration"`
	Script        string `json:"script"`
	Type          string `json:"type"`
	CodecId       string `json:"codecId"`
	CertBase64    string `json:"certBase64"` // crt文件base64
	KeyBase64     string `json:"keyBase64"`  // key文件base64
}

type NetServer interface {
	Type() NetType
	Start(n NetworkConf) error
	Reload() error
	Stop() error
	TotalConnection() int32
}

type NetClient interface {
	Type() NetType
	Connect(deviceId string, n NetworkConf) error
	Reload() error
	Close() error
}

// network meta config
type networkMetaConfig struct {
	sync.Mutex
	m map[string]func() DefaultMetaConfig
}

var defaultnetworkMetaConfig networkMetaConfig = networkMetaConfig{m: map[string]func() DefaultMetaConfig{}}

func RegNetworkMetaConfigCreator(networkType string, fn func() DefaultMetaConfig) {
	defaultnetworkMetaConfig.Lock()
	defer defaultnetworkMetaConfig.Unlock()
	defaultnetworkMetaConfig.m[networkType] = fn
}

func GetNetworkMetaConfig(networkType string) DefaultMetaConfig {
	defaultnetworkMetaConfig.Lock()
	defer defaultnetworkMetaConfig.Unlock()
	if v, ok := defaultnetworkMetaConfig.m[networkType]; ok {
		return v()
	}
	return DefaultMetaConfig{MetaConfigs: []ProductMetaConfig{}}
}
