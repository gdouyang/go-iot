package modbus

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
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

	client    MODBUS.Client
	productId string
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

func (c *ModbusClient) GetValue(typ string, startingAddress uint16, length uint16) ([]byte, error) {
	// Reading value from device
	var response []byte
	var err error

	switch typ {
	case DISCRETES_INPUT:
		response, err = c.client.ReadDiscreteInputs(startingAddress, length)
	case COILS:
		response, err = c.client.ReadCoils(startingAddress, length)

	case INPUT_REGISTERS:
		response, err = c.client.ReadInputRegisters(startingAddress, length)
	case HOLDING_REGISTERS:
		response, err = c.client.ReadHoldingRegisters(startingAddress, length)
	default:
		return response, errors.New("none supported primary table")
	}

	if err != nil {
		return nil, err
	}

	logs.Info(fmt.Sprintf("Modbus client GetValue's results %v", response))

	return response, nil
}

func (c *ModbusClient) SetValue(typ string, startingAddress uint16, length uint16, hexStr string) error {
	var err error
	value, err := hex.DecodeString(hexStr)
	if err != nil {
		logs.Error("ReadHoldingRegisters error: ", err)
		return err
	}
	// Write value to device
	var result []byte

	switch typ {
	case DISCRETES_INPUT:
		err = fmt.Errorf("error: DISCRETES_INPUT is Read-Only..!! ")
	case COILS:
		result, err = c.client.WriteMultipleCoils(startingAddress, length, value)

	case INPUT_REGISTERS:
		err = fmt.Errorf("error: INPUT_REGISTERS is Read-Only..!! ")

	case HOLDING_REGISTERS:
		if length == 1 {
			result, err = c.client.WriteSingleRegister(startingAddress, binary.BigEndian.Uint16(value))
		} else {
			result, err = c.client.WriteMultipleRegisters(startingAddress, length, value)
		}
	default:
	}

	if err != nil {
		return err
	}
	logs.Info(fmt.Sprintf("Modbus client SetValue successful, results: %v", result))

	return nil
}

func NewDeviceClient(protocol string, connectionInfo interface{}) (*ModbusClient, error) {
	client := new(ModbusClient)
	var err error
	var tcpInfo1 *TcpInfo
	var rtuInfo1 *RtuInfo
	if protocol == ProtocolTCP {
		client.IsModbusTcp = true
		tcpInfo1 = connectionInfo.(*TcpInfo)
	} else {
		rtuInfo1 = connectionInfo.(*RtuInfo)
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
