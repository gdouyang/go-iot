package modbus

import (
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/tsl"
	"strconv"
	"sync"
	"sync/atomic"
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
	stopped      atomic.Bool
	closeOnce    sync.Once
	client       *ModbusClient
	tcpInfo      *TcpInfo
	rtuInfo      *RtuInfo
	done         chan struct{}
	info         map[string]any
}

func newSession() *ModbusSession {
	return &ModbusSession{
		lock: make(chan bool, 1),
		done: make(chan struct{}),
		info: map[string]any{},
	}
}

func (s *ModbusSession) Disconnect() error {
	// API Close 与 reload/codec 关闭路径可能并发触发 Disconnect；原实现无锁读/写 stopped
	// 后 close(s.done)/close(s.lock) 会发生 close of closed channel panic。
	// sync.Once 保证 done 只 close 一次；不 close s.lock 以免与并发 lockAddress 的
	// s.lock<-true 互相击穿為 send on closed channel panic。
	s.closeOnce.Do(func() {
		s.stopped.Store(true)
		core.DelSessionByUserDisconnect(s.deviceId)
		close(s.done)
	})
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

func (s *ModbusSession) GetConInfo() map[string]any {
	return s.info
}

func (s *ModbusSession) ReadDiscreteInputs(startingAddress uint16, length uint16) (*context, error) {
	return s.getValue(DISCRETES_INPUT, startingAddress, length)
}
func (s *ModbusSession) ReadCoils(startingAddress uint16, length uint16) (*context, error) {
	return s.getValue(COILS, startingAddress, length)
}
func (s *ModbusSession) ReadInputRegisters(startingAddress uint16, length uint16) (*context, error) {
	return s.getValue(INPUT_REGISTERS, startingAddress, length)
}
func (s *ModbusSession) ReadHoldingRegisters(startingAddress uint16, length uint16) (*context, error) {
	return s.getValue(HOLDING_REGISTERS, startingAddress, length)
}

func (s *ModbusSession) getValue(parimaryTable string, startingAddress uint16, length uint16) (*context, error) {
	data, err := s.client.GetValue(parimaryTable, startingAddress, length)
	if err != nil {
		return nil, err
	}
	return &context{
		BaseContext: core.BaseContext{
			DeviceId:  s.deviceId,
			ProductId: s.productId,
			Session:   s,
		},
		Data: data,
	}, nil
}

func (s *ModbusSession) WriteCoils(startingAddress uint16, length uint16, hexStr string) error {
	return s.setValue(COILS, startingAddress, length, hexStr)
}

func (s *ModbusSession) WriteHoldingRegisters(startingAddress uint16, length uint16, hexStr string) error {
	return s.setValue(HOLDING_REGISTERS, startingAddress, length, hexStr)
}

func (s *ModbusSession) setValue(parimaryTable string, startingAddress uint16, length uint16, hexStr string) error {
	return s.client.SetValue(parimaryTable, startingAddress, length, hexStr)
}

// lockAddress mark address is unavailable because real device handle one request at a time
func (s *ModbusSession) lockAddress(address string) error {
	if s.stopped.Load() {
		return fmt.Errorf("service attempts to stop and unable to handle new request")
	}
	s.mutex.Lock()

	// workingAddressCount used to check high-frequency command execution to avoid goroutine block
	if s.workingCount == 0 {
		s.workingCount = 1
	} else if s.workingCount >= concurrentCommandLimit {
		s.mutex.Unlock()
		errorMessage := fmt.Sprintf("High-frequency command execution. There are %v commands with the same address in the queue", concurrentCommandLimit)
		logs.Errorf("%s", errorMessage)
		return fmt.Errorf("%s", errorMessage)
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

func (s *ModbusSession) interval(f tsl.Function) {
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
					core.DoCmdInvoke(core.FuncInvoke{
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
