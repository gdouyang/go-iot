package codec

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
