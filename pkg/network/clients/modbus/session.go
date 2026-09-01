package modbus

import (
	stdctx "context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"errors"

	"go-iot/pkg/core"

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
	idleOnce     sync.Once
	client       *ModbusClient
	tcpInfo      *TcpInfo
	rtuInfo      *RtuInfo
	done         chan struct{}
	info         map[string]any
	infoMu       sync.Mutex
	lastUsed     atomic.Int64
	lastCollect  atomic.Int64

	schedMu     sync.Mutex
	schedCtx    stdctx.Context
	schedCancel stdctx.CancelFunc
	schedGen    atomic.Uint64
	runtime     *CollectorRuntime
	lastValues  sync.Map
	failCount   sync.Map
	heartbeatN  sync.Map
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
		if s.schedCancel != nil {
			s.schedCancel()
		}
		unregisterSession(s.deviceId)
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

func (s *ModbusSession) debugLog(level, format string, args ...any) {
	if s == nil {
		return
	}
	core.DebugLog(level, s.deviceId, s.productId, fmt.Sprintf(format, args...))
}

func (s *ModbusSession) GetConInfo() map[string]any {
	s.infoMu.Lock()
	defer s.infoMu.Unlock()
	out := map[string]any{}
	for k, v := range s.info {
		out[k] = v
	}
	if ts := s.lastCollect.Load(); ts > 0 {
		out["lastCollect"] = ts
	}
	return out
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
	var out *context
	err := s.withConn(func(cli *ModbusClient) error {
		data, err := cli.GetValue(parimaryTable, startingAddress, length)
		if err != nil {
			return err
		}
		out = &context{
			BaseContext: core.BaseContext{
				DeviceId:  s.deviceId,
				ProductId: s.productId,
				Session:   s,
			},
			Data: data,
		}
		return nil
	})
	return out, err
}

func (s *ModbusSession) WriteCoils(startingAddress uint16, length uint16, hexStr string) error {
	return s.setValue(COILS, startingAddress, length, hexStr)
}

func (s *ModbusSession) WriteHoldingRegisters(startingAddress uint16, length uint16, hexStr string) error {
	return s.setValue(HOLDING_REGISTERS, startingAddress, length, hexStr)
}

func (s *ModbusSession) setValue(parimaryTable string, startingAddress uint16, length uint16, hexStr string) error {
	return s.withConn(func(cli *ModbusClient) error {
		return cli.SetValue(parimaryTable, startingAddress, length, hexStr)
	})
}

func (s *ModbusSession) Int16ToData(val int16) string {
	return hexEncodeBinary(val)
}

func (s *ModbusSession) FloatToInt16Data(val float64) string {
	return hexEncodeBinary(int16(val))
}

func (s *ModbusSession) FloatToUint16Data(val float64) string {
	return hexEncodeBinary(uint16(val))
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
	if s.stopped.Load() {
		s.unlockAddress(address)
		return fmt.Errorf("service attempts to stop and unable to handle new request")
	}
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
	return s.withConn(func(_ *ModbusClient) error {
		if callback != nil {
			callback()
		}
		return nil
	})
}

func (s *ModbusSession) withConn(fn func(*ModbusClient) error) error {
	var connectionInfo interface{} = s.tcpInfo
	if s.rtuInfo != nil {
		connectionInfo = s.rtuInfo
	}
	if err := s.lockAddress(s.lockableAddress(connectionInfo)); err != nil {
		return err
	}
	defer s.unlockAddress(s.lockableAddress(connectionInfo))

	protocol := ProtocolTCP
	if s.rtuInfo != nil {
		protocol = ProtocolRTU
	}

	open := func() error {
		if s.client != nil {
			return nil
		}
		deviceClient, err := NewDeviceClient(protocol, connectionInfo)
		if err != nil {
			logs.Errorf("Read command NewDeviceClient failed. err:%v \n", err)
			s.debugLog("error", "collector NewDeviceClient failed: %v", err)
			return err
		}
		if err = deviceClient.OpenConnection(); err != nil {
			logs.Errorf("Read command OpenConnection failed. err:%v \n", err)
			s.debugLog("error", "collector OpenConnection failed: %v", err)
			return err
		}
		s.client = deviceClient
		return nil
	}
	closeNow := func() {
		if s.client != nil {
			_ = s.client.CloseConnection()
			s.client = nil
		}
	}

	run := func() error {
		if err := open(); err != nil {
			return err
		}
		err := fn(s.client)
		if err != nil && !errors.Is(err, core.ErrFunctionNotImpl) {
			closeNow()
		}
		return err
	}

	err := run()
	if err != nil && !errors.Is(err, core.ErrFunctionNotImpl) {
		err = run()
	}
	if err == nil || errors.Is(err, core.ErrFunctionNotImpl) {
		s.lastUsed.Store(time.Now().UnixMilli())
		s.startIdleLoop()
	}
	if s.stopped.Load() {
		closeNow()
	}
	return err
}

func (s *ModbusSession) startIdleLoop() {
	s.idleOnce.Do(func() {
		go s.idleLoop()
	})
}

func (s *ModbusSession) idleTimeout() time.Duration {
	sec := 5
	if s.tcpInfo != nil && s.tcpInfo.IdleTimeout > 0 {
		sec = s.tcpInfo.IdleTimeout
	} else if s.rtuInfo != nil && s.rtuInfo.IdleTimeout > 0 {
		sec = s.rtuInfo.IdleTimeout
	}
	return time.Duration(sec) * time.Second
}

func (s *ModbusSession) idleLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			s.closeHandlerIfIdle()
			return
		case <-ticker.C:
			if s.client == nil {
				continue
			}
			if time.Since(time.UnixMilli(s.lastUsed.Load())) >= s.idleTimeout() {
				s.closeHandlerIfIdle()
			}
		}
	}
}

func (s *ModbusSession) closeHandlerIfIdle() {
	s.lock <- true
	defer func() { <-s.lock }()
	if !s.stopped.Load() {
		if time.Since(time.UnixMilli(s.lastUsed.Load())) < s.idleTimeout() {
			return
		}
	}
	if s.client != nil {
		_ = s.client.CloseConnection()
		s.client = nil
	}
}

func (s *ModbusSession) readLoop() {
	s.startSchedulers()
}
