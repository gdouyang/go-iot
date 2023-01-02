package modbus

import (
	"encoding/json"
	"fmt"
	"go-iot/codec"
	"strconv"
)

type ModbusSpec struct {
	Protocol string `json:"protocol"`
	Conf     string `json:"conf"`
}

func (s *ModbusSpec) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), s)
	if err != nil {
		return fmt.Errorf("modbusSpec FromJson error: %v", err)
	}
	return nil
}

func (s *ModbusSpec) SetTcpByConfig(devoper *codec.Device) error {
	errorMessage := "unable to create TCP connection info, protocol config '%s' not exist"
	address := devoper.GetConfig("address")
	if len(address) == 0 {
		return fmt.Errorf(errorMessage, "address")
	}
	s.Protocol = ProtocolTCP
	info := TcpInfo{
		Address: address,
	}
	if len(devoper.GetConfig("port")) == 0 {
		return fmt.Errorf(errorMessage, "port")
	}
	port, err := strconv.ParseUint(devoper.GetConfig("port"), 0, 16)
	if err != nil {
		return fmt.Errorf("port value out of range(0–65535). Error: %v", err)
	}
	info.Port = int(port)
	if len(devoper.GetConfig("unitID")) == 0 {
		return fmt.Errorf(errorMessage, "unitID")
	}
	unitID, err := strconv.ParseUint(devoper.GetConfig("unitID"), 0, 8)
	if err != nil {
		return fmt.Errorf("uintID value out of range(0–255). Error: %v", err)
	}
	info.UnitID = byte(unitID)
	timeout, err := parseIntValue(devoper.GetConfig("timeout"), "timeout")
	if err != nil {
		return err
	}
	info.Timeout = timeout
	idleTimeout, err := parseIntValue(devoper.GetConfig("idleTimeout"), "idleTimeout")
	if err != nil {
		return err
	}
	info.IdleTimeout = idleTimeout
	b, _ := json.Marshal(info)
	s.Conf = string(b)
	return nil
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
	BaudRate int    `json:"baudRate"`
	DataBits int    `json:"dataBits"`
	StopBits int    `json:"stopBits"`
	Parity   string `json:"parity"`
	UnitID   uint8  `json:"unitID"`
	// Connect & Read timeout(seconds)
	Timeout int `json:"timeout"`
	// Idle timeout(seconds) to close the connection
	IdleTimeout int `json:"idleTimeout"`
}

func createRTUConnectionInfo(rtuProtocol string) (info *RtuInfo, err error) {
	err = json.Unmarshal([]byte(rtuProtocol), info)
	if err != nil {
		return nil, err
	}

	if info.Parity != "N" && info.Parity != "O" && info.Parity != "E" {
		return nil, fmt.Errorf("invalid parity value, it should be N(None) or O(Odd) or E(Even)")
	}

	return info, nil
}

func createTcpConnectionInfo(tcpProtocol string) (info *TcpInfo, err error) {
	err = json.Unmarshal([]byte(tcpProtocol), info)
	if err != nil {
		return nil, err
	}
	if info.Port < 0 {
		return nil, fmt.Errorf("port value out of range(0–65535). ")
	}

	return info, nil
}
