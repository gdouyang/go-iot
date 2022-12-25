package modbus

import (
	"fmt"
	"go-iot/codec"
	"sync"

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
}

func newSession() *session {
	return &session{
		lock: make(chan bool, 1),
	}
}

func (s *session) Disconnect() error {
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
func (d *session) lockAddress(address string) error {
	if d.stopped {
		return fmt.Errorf("service attempts to stop and unable to handle new request")
	}
	d.mutex.Lock()

	// workingAddressCount used to check high-frequency command execution to avoid goroutine block
	if d.workingCount == 0 {
		d.workingCount = 1
	} else if d.workingCount >= concurrentCommandLimit {
		d.mutex.Unlock()
		errorMessage := fmt.Sprintf("High-frequency command execution. There are %v commands with the same address in the queue", concurrentCommandLimit)
		logs.Error(errorMessage)
		return fmt.Errorf(errorMessage)
	} else {
		d.workingCount = d.workingCount + 1
	}

	d.mutex.Unlock()
	d.lock <- true

	return nil
}

// unlockAddress remove token after command finish
func (d *session) unlockAddress(address string) {
	d.mutex.Lock()
	d.workingCount = d.workingCount - 1
	d.mutex.Unlock()
	<-d.lock
}

// lockableAddress return the lockable address according to the protocol
func (d *session) lockableAddress(info interface{}) string {
	var tcpInfo1 *tcpInfo
	var rtuInfo1 *rtuInfo
	var address string
	if d.protocol == ProtocolTCP {
		address = fmt.Sprintf("%s:%d", tcpInfo1.Address, tcpInfo1.Port)
	} else {
		rtuInfo1 = info.(*rtuInfo)
		address = rtuInfo1.Address
	}
	return address
}

func (d *session) connection(callback func()) error {
	var connectionInfo interface{}
	var err error
	if d.protocol == ProtocolTCP {
		connectionInfo, err = createTcpConnectionInfo(d.conf)
		if err != nil {
			logs.Error(err)
			return err
		}
	} else {
		connectionInfo, err = createRTUConnectionInfo(d.conf)
		if err != nil {
			logs.Error(err)
			return err
		}
	}

	err = d.lockAddress(d.lockableAddress(connectionInfo))
	if err != nil {
		return err
	}
	defer d.unlockAddress(d.lockableAddress(connectionInfo))

	// create device client and open connection
	deviceClient, err := NewDeviceClient(d.protocol, connectionInfo)
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
	d.client = deviceClient
	callback()
	return nil
}
