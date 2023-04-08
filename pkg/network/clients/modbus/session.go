package modbus

import (
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/common"
	"go-iot/pkg/core/tsl"
	"strconv"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

var concurrentCommandLimit = 100

type modbusSession struct {
	deviceId     string
	productId    string
	mutex        sync.Mutex
	lock         chan bool
	workingCount int
	stopped      bool
	client       *ModbusClient
	tcpInfo      *TcpInfo
	rtuInfo      *RtuInfo
	done         chan struct{}
}

func newSession() *modbusSession {
	return &modbusSession{
		lock: make(chan bool, 1),
		done: make(chan struct{}),
	}
}

func (s *modbusSession) Disconnect() error {
	if !s.stopped {
		core.DelSession(s.deviceId)
		s.stopped = true
		close(s.done)
		close(s.lock)
	}
	return nil
}

func (s *modbusSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *modbusSession) GetDeviceId() string {
	return s.deviceId
}

func (s *modbusSession) ReadDiscreteInputs(startingAddress uint16, length uint16) *context {
	return s.getValue(DISCRETES_INPUT, startingAddress, length)
}
func (s *modbusSession) ReadCoils(startingAddress uint16, length uint16) *context {
	return s.getValue(COILS, startingAddress, length)
}
func (s *modbusSession) ReadInputRegisters(startingAddress uint16, length uint16) *context {
	return s.getValue(INPUT_REGISTERS, startingAddress, length)
}
func (s *modbusSession) ReadHoldingRegisters(startingAddress uint16, length uint16) *context {
	return s.getValue(HOLDING_REGISTERS, startingAddress, length)
}

func (s *modbusSession) getValue(parimaryTable string, startingAddress uint16, length uint16) *context {
	data, err := s.client.GetValue(parimaryTable, startingAddress, length)
	if err != nil {
		panic(err)
	}
	return &context{
		BaseContext: core.BaseContext{
			DeviceId:  s.deviceId,
			ProductId: s.productId,
			Session:   s,
		},
		Data: data,
	}
}

func (s *modbusSession) WriteCoils(startingAddress uint16, length uint16, hexStr string) {
	s.setValue(COILS, startingAddress, length, hexStr)
}

func (s *modbusSession) WriteHoldingRegisters(startingAddress uint16, length uint16, hexStr string) {
	s.setValue(HOLDING_REGISTERS, startingAddress, length, hexStr)
}

func (s *modbusSession) setValue(parimaryTable string, startingAddress uint16, length uint16, hexStr string) {
	err := s.client.SetValue(parimaryTable, startingAddress, length, hexStr)
	if err != nil {
		panic(err)
	}
}

// lockAddress mark address is unavailable because real device handle one request at a time
func (s *modbusSession) lockAddress(address string) error {
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
func (s *modbusSession) unlockAddress(address string) {
	s.mutex.Lock()
	s.workingCount = s.workingCount - 1
	s.mutex.Unlock()
	<-s.lock
}

// lockableAddress return the lockable address according to the protocol
func (s *modbusSession) lockableAddress(info interface{}) string {
	var address string
	if s.tcpInfo != nil {
		address = fmt.Sprintf("%s:%d", s.tcpInfo.Address, s.tcpInfo.Port)
	} else {
		address = s.rtuInfo.Address
	}
	return address
}

func (s *modbusSession) connection(callback func()) error {
	var connectionInfo interface{} = s.tcpInfo
	var err error
	if s.rtuInfo != nil {
		connectionInfo = s.rtuInfo
	}

	err = s.lockAddress(s.lockableAddress(connectionInfo))
	if err != nil {
		return err
	}
	defer s.unlockAddress(s.lockableAddress(connectionInfo))

	// create device client and open connection
	var protocol string = ProtocolTCP
	if s.tcpInfo != nil {
		protocol = ProtocolTCP
	}
	deviceClient, err := NewDeviceClient(protocol, connectionInfo)
	if err != nil {
		logs.Error("Read command NewDeviceClient failed. err:%v \n", err)
		return err
	}

	err = deviceClient.OpenConnection()
	if err != nil {
		logs.Error("Read command OpenConnection failed. err:%v \n", err)
		return err
	}

	defer func() {
		_ = deviceClient.CloseConnection()
		s.client = nil
	}()
	s.client = deviceClient
	callback()
	return nil
}

func (s *modbusSession) readLoop() {
	product := core.GetProduct(s.productId)
	if product != nil {
		for _, f := range product.GetTsl().Functions {
			go s.interval(f)
		}
	}
}

func (s *modbusSession) interval(f tsl.TslFunction) {
	if f.Expands != nil {
		if val, ok := f.Expands["interval"]; ok && len(val) > 0 {
			num, err := strconv.Atoi(val)
			if err != nil {
				logs.Warn("interval must gt 0, error:", err)
				return
			}
			if num < 1 {
				logs.Warn("interval must gt 0, function=", f.Id)
				return
			}
			for {
				select {
				case <-time.After(time.Second * time.Duration(num)):
					core.DoCmdInvoke(common.FuncInvoke{
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
