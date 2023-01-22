package codec

import "sync"

// 服务器类型
type NetServerType string

// 客户端类型
type NetClientType string

const (
	// MQTT服务端
	MQTT_BROKER NetServerType = "MQTT_BROKER"
	// TCP服务端
	TCP_SERVER NetServerType = "TCP_SERVER"
	// HTTP服务端
	HTTP_SERVER NetServerType = "HTTP_SERVER"
	// WebSocket服务端
	WEBSOCKET_SERVER NetServerType = "WEBSOCKET_SERVER"

	// MQTT客户端
	MQTT_CLIENT NetClientType = "MQTT_CLIENT"
	// TCP客户端
	TCP_CLIENT NetClientType = "TCP_CLIENT"
	// MODBUS
	MODBUS NetClientType = "MODBUS"
)

func IsNetClientType(str string) bool {
	return TCP_CLIENT == NetClientType(str) || MQTT_CLIENT == NetClientType(str) || MODBUS == NetClientType(str)
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
	Type() NetServerType
	Start(n NetworkConf) error
	Reload() error
	Stop() error
	TotalConnection() int32
}

type NetClient interface {
	Type() NetClientType
	Connect(deviceId string, n NetworkConf) error
	Reload() error
	Close() error
}

type networkMetaConfig struct {
	sync.Mutex
	m map[string]func() ProductMetaConfigs
}

var defaultnetworkMetaConfig networkMetaConfig = networkMetaConfig{m: map[string]func() ProductMetaConfigs{}}

func RegNetworkMetaConfigCreator(networkType string, fn func() ProductMetaConfigs) {
	defaultnetworkMetaConfig.Lock()
	defer defaultnetworkMetaConfig.Unlock()
	defaultnetworkMetaConfig.m[networkType] = fn
}
func GetNetworkMetaConfig(networkType string) ProductMetaConfigs {
	defaultnetworkMetaConfig.Lock()
	defer defaultnetworkMetaConfig.Unlock()
	if v, ok := defaultnetworkMetaConfig.m[networkType]; ok {
		return v()
	}
	return ProductMetaConfigs{}
}
