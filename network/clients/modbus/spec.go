package modbus

import (
	"encoding/json"
	"fmt"
)

type modbusSpec struct {
	Protocol string `json:"protocol"`
	Conf     string `json:"conf"`
}

func (s *modbusSpec) FromJson(str string) error {
	err := json.Unmarshal([]byte(str), s)
	if err != nil {
		return fmt.Errorf("modbusSpec FromJson error: %v", err)
	}
	return nil
}

type tcpInfo struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	UnitID  uint8  `json:"unitID"`
	// Connect & Read timeout(seconds)
	Timeout int `json:"timeout"`
	// Idle timeout(seconds) to close the connection
	IdleTimeout int `json:"idleTimeout"`
}

type rtuInfo struct {
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

func createRTUConnectionInfo(rtuProtocol string) (info *rtuInfo, err error) {
	err = json.Unmarshal([]byte(rtuProtocol), info)
	if err != nil {
		return nil, err
	}

	if info.Parity != "N" && info.Parity != "O" && info.Parity != "E" {
		return nil, fmt.Errorf("invalid parity value, it should be N(None) or O(Odd) or E(Even)")
	}

	return info, nil
}

func createTcpConnectionInfo(tcpProtocol string) (info *tcpInfo, err error) {
	err = json.Unmarshal([]byte(tcpProtocol), info)
	if err != nil {
		return nil, err
	}
	if info.Port < 0 {
		return nil, fmt.Errorf("port value out of range(0–65535). ")
	}

	return info, nil
}
