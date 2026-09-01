package modbus

import (
	"encoding/json"
	"fmt"
	"strconv"

	"go-iot/pkg/core"
	"go-iot/pkg/network"
)

func init() {
	network.RegNetworkMetaConfigCreator(string(network.MODBUS), func() core.CodecMetaConfig {

		list := []core.MetaConfig{
			{Property: "address", Type: "string", Buildin: true, Value: "127.0.0.1", Desc: "从站地址，例如 127.0.0.1"},
			{Property: "port", Type: "number", Buildin: true, Value: "502", Desc: "从站端口，默认 502"},
			{Property: "unitID", Type: "number", Buildin: true, Desc: "从站号（Unit ID）。TCP 写在报文头，用来指定连哪一台从站。范围 1–247，不能为空。"},
			{Property: "timeout", Type: "number", Buildin: true, Value: "5", Desc: "连接和读超时（秒）"},
			{Property: "idleTimeout", Type: "number", Buildin: true, Value: "5", Desc: "空闲多久关闭 TCP 连接（秒）"},

			// {Property: "baudRate", Type: "number", Buildin: true, Desc: ""},
			// {Property: "dataBits", Type: "number", Buildin: true, Desc: ""},
			// {Property: "stopBits", Type: "number", Buildin: true, Desc: ""},
			// {Property: "parity", Type: "number", Buildin: true, Desc: ""},
		}
		return core.CodecMetaConfig{MetaConfigs: list}
	})
}

func parseIntValue(str string, key string) (int, error) {
	if len(str) == 0 {
		return 5, nil
	}
	val, err := strconv.Atoi(str)
	if err != nil {
		return 0, fmt.Errorf("fail to parse protocol config '%s', %v", key, err)
	}
	return val, nil
}

type TcpInfo struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	UnitID  uint8  `json:"unitID"`
	// Connect & Read timeout(seconds)
	Timeout int `json:"timeout"`
	// Idle timeout(seconds) to close the connection
	IdleTimeout int `json:"idleTimeout"`
}

type RtuInfo struct {
	Address  string `json:"address"`
	BaudRate int    `json:"baudRate"` // 波特率
	DataBits int    `json:"dataBits"`
	StopBits int    `json:"stopBits"`
	Parity   string `json:"parity"`
	UnitID   uint8  `json:"unitID"`
	// Connect & Read timeout(seconds)
	Timeout int `json:"timeout"`
	// Idle timeout(seconds) to close the connection
	IdleTimeout int `json:"idleTimeout"`
}

func createRTUConnectionInfo(rtuProtocol string) (*RtuInfo, error) {
	info := &RtuInfo{}
	err := json.Unmarshal([]byte(rtuProtocol), info)
	if err != nil {
		return nil, err
	}

	if info.Parity != "N" && info.Parity != "O" && info.Parity != "E" {
		return nil, fmt.Errorf("invalid parity value, it should be N(None) or O(Odd) or E(Even)")
	}

	return info, nil
}

func createTcpConnectionInfo(tcpProtocol string) (*TcpInfo, error) {
	info := &TcpInfo{}
	err := json.Unmarshal([]byte(tcpProtocol), info)
	if err != nil {
		return nil, err
	}
	if info.Port < 0 {
		return nil, fmt.Errorf("port value out of range(0–65535). ")
	}

	return info, nil
}

func createTcpConnectionInfoByConfig(devoper *core.Device) (*TcpInfo, error) {
	errorMessage := "unable to create TCP connection info, protocol config '%s' not exist"
	address := devoper.GetConfig("address")
	if len(address) == 0 {
		return nil, fmt.Errorf(errorMessage, "address")
	}
	info := TcpInfo{
		Address: address,
	}
	if len(devoper.GetConfig("port")) == 0 {
		return nil, fmt.Errorf(errorMessage, "port")
	}
	port, err := strconv.ParseUint(devoper.GetConfig("port"), 0, 16)
	if err != nil {
		return nil, fmt.Errorf("port value out of range(0–65535). Error: %v", err)
	}
	info.Port = int(port)
	if len(devoper.GetConfig("unitID")) == 0 {
		return nil, fmt.Errorf(errorMessage, "unitID")
	}
	unitID, err := strconv.ParseUint(devoper.GetConfig("unitID"), 0, 8)
	if err != nil {
		return nil, fmt.Errorf("uintID value out of range(0–255). Error: %v", err)
	}
	info.UnitID = byte(unitID)
	timeout, err := parseIntValue(devoper.GetConfig("timeout"), "timeout")
	if err != nil {
		return nil, err
	}
	info.Timeout = timeout
	idleTimeout, err := parseIntValue(devoper.GetConfig("idleTimeout"), "idleTimeout")
	if err != nil {
		return nil, err
	}
	info.IdleTimeout = idleTimeout
	return &info, nil
}
