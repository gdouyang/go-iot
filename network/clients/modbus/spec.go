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
	Address string
	Port    int
	UnitID  uint8
	// Connect & Read timeout(seconds)
	Timeout int
	// Idle timeout(seconds) to close the connection
	IdleTimeout int
}

type rtuInfo struct {
	Address  string
	BaudRate int
	DataBits int
	StopBits int
	Parity   string
	UnitID   uint8
	// Connect & Read timeout(seconds)
	Timeout int
	// Idle timeout(seconds) to close the connection
	IdleTimeout int
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
