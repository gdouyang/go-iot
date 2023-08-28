package modbus

import (
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/core/common"
	"go-iot/pkg/core/tsl"
	"strconv"
	"sync"
	"time"

	logs "go-iot/pkg/logger"
)

var concurrentCommandLimit = 100

// modbus协议Session
type ModbusSession struct {
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

func newSession() *ModbusSession {
	return &ModbusSession{
		lock: make(chan bool, 1),
		done: make(chan struct{}),
	}
}

func (s *ModbusSession) Disconnect() error {
	if !s.stopped {
		core.DelSession(s.deviceId)
		s.stopped = true
		close(s.done)
		close(s.lock)
	}
	return nil
}

func (s *ModbusSession) Close() error {
	return s.Disconnect()
}
func (s *ModbusSession) SetDeviceId(deviceId string) {
	s.deviceId = deviceId
}

func (s *ModbusSession) GetDeviceId() string {
	return s.deviceId
}

func (s *ModbusSession) ReadDiscreteInputs(startingAddress uint16, length uint16) *context {
	return s.getValue(DISCRETES_INPUT, startingAddress, length)
}
func (s *ModbusSession) ReadCoils(startingAddress uint16, length uint16) *context {
	return s.getValue(COILS, startingAddress, length)
}
func (s *ModbusSession) ReadInputRegisters(startingAddress uint16, length uint16) *context {
	return s.getValue(INPUT_REGISTERS, startingAddress, length)
}
func (s *ModbusSession) ReadHoldingRegisters(startingAddress uint16, length uint16) *context {
	return s.getValue(HOLDING_REGISTERS, startingAddress, length)
}

func (s *ModbusSession) getValue(parimaryTable string, startingAddress uint16, length uint16) *context {
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

func (s *ModbusSession) WriteCoils(startingAddress uint16, length uint16, hexStr string) {
	s.setValue(COILS, startingAddress, length, hexStr)
}

func (s *ModbusSession) WriteHoldingRegisters(startingAddress uint16, length uint16, hexStr string) {
	s.setValue(HOLDING_REGISTERS, startingAddress, length, hexStr)
}

func (s *ModbusSession) setValue(parimaryTable string, startingAddress uint16, length uint16, hexStr string) {
	err := s.client.SetValue(parimaryTable, startingAddress, length, hexStr)
	if err != nil {
		panic(err)
	}
}

// lockAddress mark address is unavailable because real device handle one request at a time
func (s *ModbusSession) lockAddress(address string) error {
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
		logs.Errorf(errorMessage)
		return fmt.Errorf(errorMessage)
	} else {
		s.workingCount = s.workingCount + 1
	}

	s.mutex.Unlock()
	s.lock <- true

	return nil
}

// unlockAddress remove token after command finish
func (s *ModbusSession) unlockAddress(address string) {
	s.mutex.Lock()
	s.workingCount = s.workingCount - 1
	s.mutex.Unlock()
	<-s.lock
}

// lockableAddress return the lockable address according to the protocol
func (s *ModbusSession) lockableAddress(info interface{}) string {
	var address string
	if s.tcpInfo != nil {
		address = fmt.Sprintf("%s:%d", s.tcpInfo.Address, s.tcpInfo.Port)
	} else {
		address = s.rtuInfo.Address
	}
	return address
}

func (s *ModbusSession) connection(callback func()) error {
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
		logs.Errorf("Read command NewDeviceClient failed. err:%v \n", err)
		return err
	}

	err = deviceClient.OpenConnection()
	if err != nil {
		logs.Errorf("Read command OpenConnection failed. err:%v \n", err)
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

func (s *ModbusSession) readLoop() {
	product := core.GetProduct(s.productId)
	if product != nil {
		for _, f := range product.GetTsl().Functions {
			go s.interval(f)
		}
	}
}

func (s *ModbusSession) interval(f tsl.TslFunction) {
	if f.Expands != nil {
		if val, ok := f.Expands["interval"]; ok && len(val) > 0 {
			num, err := strconv.Atoi(val)
			if err != nil {
				logs.Warnf("interval must gt 0, error: %v", err)
				return
			}
			if num < 1 {
				logs.Warnf("interval must gt 0, function=%v", f.Id)
				return
			}
			ticker := time.NewTicker(time.Second * time.Duration(num))
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
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
