package modbus

import (
	"fmt"
	"go-iot/codec"
	"go-iot/codec/msg"
	"go-iot/codec/tsl"
	"strconv"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var concurrentCommandLimit = 100

type session struct {
	deviceId     string
	productId    string
	mutex        sync.Mutex
	lock         chan bool
	workingCount int
	stopped      bool
	client       *ModbusClient
	protocol     string
	conf         string
	done         chan struct{}
}

func newSession() *session {
	return &session{
		lock: make(chan bool, 1),
		done: make(chan struct{}),
	}
}

func (s *session) Disconnect() error {
	if !s.stopped {
		codec.DelSession(s.deviceId)
		s.stopped = true
		close(s.done)
	}
	return nil
}

func (s *session) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *session) GetDeviceId() string {
	return s.deviceId
}

func (s *session) ReadDiscreteInputs(startingAddress uint16, length uint16) {
	s.getValue(DISCRETES_INPUT, startingAddress, length)
}
func (s *session) ReadCoils(startingAddress uint16, length uint16) {
	s.getValue(COILS, startingAddress, length)
}
func (s *session) ReadInputRegisters(startingAddress uint16, length uint16) {
	s.getValue(INPUT_REGISTERS, startingAddress, length)
}
func (s *session) ReadHoldingRegisters(startingAddress uint16, length uint16) {
	s.getValue(HOLDING_REGISTERS, startingAddress, length)
}

func (s *session) getValue(typ string, startingAddress uint16, length uint16) {
	s.connection(func() {
		data, err := s.client.GetValue(typ, startingAddress, length)
		if err != nil {
			return
		}
		cod := codec.GetCodec(s.productId)
		if cod != nil {
			cod.OnMessage(&context{
				BaseContext: codec.BaseContext{
					DeviceId:  s.deviceId,
					ProductId: s.productId,
					Session:   s,
				},
				Data: data,
			})
		}
	})
}

func (s *session) WriteCoils(startingAddress uint16, length uint16, hexStr string) {
	s.setValue(COILS, startingAddress, length, hexStr)
}

func (s *session) WriteHoldingRegisters(startingAddress uint16, length uint16, hexStr string) {
	s.setValue(HOLDING_REGISTERS, startingAddress, length, hexStr)
}

func (s *session) setValue(typ string, startingAddress uint16, length uint16, hexStr string) {
	s.connection(func() {
		s.client.SetValue(typ, startingAddress, length, hexStr)
	})
}

// lockAddress mark address is unavailable because real device handle one request at a time
func (s *session) lockAddress(address string) error {
	if s.stopped {
		return fmt.Errorf("service attempts to stop and unable to handle new request")
	}
	s.mutex.Lock()

	// workingAddressCount used to check high-frequency command execution to avoid goroutine block
	if s.workingCount == 0 {
		s.workingCount = 1
	} else if s.workingCount >= concurrentCommandLimit {
		s.mutex.Unlock()
		errorMessage := fmt.Sprintf("High-frequency command execution. There are %v commands with the same address in the queue", concurrentCommandLimit)
		logs.Error(errorMessage)
		return fmt.Errorf(errorMessage)
	} else {
		s.workingCount = s.workingCount + 1
	}

	s.mutex.Unlock()
	s.lock <- true

	return nil
}

// unlockAddress remove token after command finish
func (s *session) unlockAddress(address string) {
	s.mutex.Lock()
	s.workingCount = s.workingCount - 1
	s.mutex.Unlock()
	<-s.lock
}

// lockableAddress return the lockable address according to the protocol
func (s *session) lockableAddress(info interface{}) string {
	var tcpInfo1 *tcpInfo
	var rtuInfo1 *rtuInfo
	var address string
	if s.protocol == ProtocolTCP {
		address = fmt.Sprintf("%s:%d", tcpInfo1.Address, tcpInfo1.Port)
	} else {
		rtuInfo1 = info.(*rtuInfo)
		address = rtuInfo1.Address
	}
	return address
}

func (s *session) connection(callback func()) error {
	var connectionInfo interface{}
	var err error
	if s.protocol == ProtocolTCP {
		connectionInfo, err = createTcpConnectionInfo(s.conf)
		if err != nil {
			logs.Error(err)
			return err
		}
	} else {
		connectionInfo, err = createRTUConnectionInfo(s.conf)
		if err != nil {
			logs.Error(err)
			return err
		}
	}

	err = s.lockAddress(s.lockableAddress(connectionInfo))
	if err != nil {
		return err
	}
	defer s.unlockAddress(s.lockableAddress(connectionInfo))

	// create device client and open connection
	deviceClient, err := NewDeviceClient(s.protocol, connectionInfo)
	if err != nil {
		logs.Error("Read command NewDeviceClient failed. err:%v \n", err)
		return err
	}

	err = deviceClient.OpenConnection()
	if err != nil {
		logs.Error("Read command OpenConnection failed. err:%v \n", err)
		return err
	}

	defer func() { _ = deviceClient.CloseConnection() }()
	s.client = deviceClient
	callback()
	return nil
}

func (s *session) readLoop() {
	product := codec.GetProduct(s.productId)
	if product != nil {
		for _, f := range product.GetTsl().Functions {
			go s.interval(f)
		}
	}
}

func (s *session) interval(f tsl.TslFunction) {
	if f.Expands != nil {
		if val, ok := f.Expands["interval"]; ok {
			num, err := strconv.Atoi(val)
			if err != nil {
				logs.Error("interval must gt 0, error:", err)
				return
			}
			if num < 1 {
				logs.Warn("interval must gt 0, function=", f.Id)
				return
			}
			for {
				select {
				case <-time.After(time.Second * time.Duration(num)):
					codec.DoCmdInvoke(s.productId, msg.FuncInvoke{
						FunctionId: f.Id,
						DeviceId:   s.deviceId,
					})
				case <-s.done:
					return
				}
			}
		}
	}
}
