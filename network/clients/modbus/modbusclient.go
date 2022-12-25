package modbus

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/beego/beego/v2/core/logs"
	MODBUS "github.com/goburrow/modbus"
)

// ModbusClient is used for connecting the device and read/write value
type ModbusClient struct {
	// IsModbusTcp is a value indicating the connection type
	IsModbusTcp bool
	// TCPClientHandler is ued for holding device TCP connection
	TCPClientHandler MODBUS.TCPClientHandler
	// TCPClientHandler is ued for holding device RTU connection
	RTUClientHandler MODBUS.RTUClientHandler

	client MODBUS.Client
}

func (c *ModbusClient) OpenConnection() error {
	var err error
	var newClient MODBUS.Client
	if c.IsModbusTcp {
		err = c.TCPClientHandler.Connect()
		newClient = MODBUS.NewClient(&c.TCPClientHandler)
		logs.Info("Modbus client create TCP connection.")
	} else {
		err = c.RTUClientHandler.Connect()
		newClient = MODBUS.NewClient(&c.RTUClientHandler)
		logs.Info("Modbus client create RTU connection.")
	}
	c.client = newClient
	return err
}

func (c *ModbusClient) CloseConnection() error {
	var err error
	if c.IsModbusTcp {
		err = c.TCPClientHandler.Close()

	} else {
		err = c.RTUClientHandler.Close()
	}
	return err
}

func (s *ModbusClient) ReadDiscreteInputs(startingAddress uint16, length uint16) string {
	response, err := s.client.ReadDiscreteInputs(startingAddress, length)
	if err != nil {
		logs.Error("ReadDiscreteInputs error: ", err)
	}
	str := hex.EncodeToString(response)
	return str
}
func (s *ModbusClient) ReadCoils(startingAddress uint16, length uint16) string {
	response, err := s.client.ReadCoils(startingAddress, length)
	if err != nil {
		logs.Error("ReadCoils error: ", err)
	}
	str := hex.EncodeToString(response)
	return str
}
func (s *ModbusClient) ReadInputRegisters(startingAddress uint16, length uint16) string {
	response, err := s.client.ReadInputRegisters(startingAddress, length)
	if err != nil {
		logs.Error("ReadInputRegisters error: ", err)
	}
	str := hex.EncodeToString(response)
	return str
}
func (s *ModbusClient) ReadHoldingRegisters(startingAddress uint16, length uint16) string {
	response, err := s.client.ReadHoldingRegisters(startingAddress, length)
	if err != nil {
		logs.Error("ReadHoldingRegisters error: ", err)
	}
	str := hex.EncodeToString(response)
	return str
}

func (s *ModbusClient) WriteCoils(startingAddress uint16, length uint16, hexStr string) {
	value, err := hex.DecodeString(hexStr)
	if err != nil {
		logs.Error("ReadHoldingRegisters error: ", err)
		return
	}
	result, err := s.client.WriteMultipleCoils(startingAddress, length, value)
	if err != nil {
		logs.Error("ReadHoldingRegisters error: ", err)
		return
	}
	logs.Info(fmt.Sprintf("Modbus client SetValue successful, results: %v", result))
}

func (s *ModbusClient) WriteHoldingRegisters(startingAddress uint16, length uint16, hexStr string) {
	var result []byte
	var err error
	value, err := hex.DecodeString(hexStr)
	if err != nil {
		logs.Error("ReadHoldingRegisters error: ", err)
		return
	}
	if length == 1 {
		result, err = s.client.WriteSingleRegister(startingAddress, binary.BigEndian.Uint16(value))
		if err != nil {
			logs.Error("WriteSingleRegister error: ", err)
			return
		}
	} else {
		result, err = s.client.WriteMultipleRegisters(startingAddress, length, value)
		if err != nil {
			logs.Error("WriteMultipleRegisters error: ", err)
			return
		}
	}
	logs.Info(fmt.Sprintf("Modbus client SetValue successful, results: %v", result))
}

func NewDeviceClient(protocol string, connectionInfo interface{}) (*ModbusClient, error) {
	client := new(ModbusClient)
	var err error
	var tcpInfo1 *tcpInfo
	var rtuInfo1 *rtuInfo
	if protocol == ProtocolTCP {
		client.IsModbusTcp = true
		tcpInfo1 = connectionInfo.(*tcpInfo)
	} else {
		rtuInfo1 = connectionInfo.(*rtuInfo)
	}
	if client.IsModbusTcp {
		client.TCPClientHandler.Address = fmt.Sprintf("%s:%d", tcpInfo1.Address, tcpInfo1.Port)
		client.TCPClientHandler.SlaveId = byte(tcpInfo1.UnitID)
		client.TCPClientHandler.Timeout = time.Duration(tcpInfo1.Timeout) * time.Second
		client.TCPClientHandler.IdleTimeout = time.Duration(tcpInfo1.IdleTimeout) * time.Second
		client.TCPClientHandler.Logger = log.New(os.Stdout, "", log.LstdFlags)
	} else {
		serialParams := strings.Split(rtuInfo1.Address, ",")
		client.RTUClientHandler.Address = serialParams[0]
		client.RTUClientHandler.SlaveId = byte(rtuInfo1.UnitID)
		client.RTUClientHandler.Timeout = time.Duration(rtuInfo1.Timeout) * time.Second
		client.RTUClientHandler.IdleTimeout = time.Duration(rtuInfo1.IdleTimeout) * time.Second
		client.RTUClientHandler.BaudRate = rtuInfo1.BaudRate
		client.RTUClientHandler.DataBits = rtuInfo1.DataBits
		client.RTUClientHandler.StopBits = rtuInfo1.StopBits
		client.RTUClientHandler.Parity = rtuInfo1.Parity
		client.RTUClientHandler.Logger = log.New(os.Stdout, "", log.LstdFlags)
	}
	return client, err
}
