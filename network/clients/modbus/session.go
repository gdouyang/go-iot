package modbus

import (
	"fmt"
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

func (s *session) ReadDiscreteInputs(startingAddress uint16, length uint16) string {
	var resp string
	s.connection(func() {
		resp = s.client.ReadDiscreteInputs(startingAddress, length)
	})
	return resp
}
func (s *session) ReadCoils(startingAddress uint16, length uint16) string {
	var resp string
	s.connection(func() {
		resp = s.client.ReadCoils(startingAddress, length)
	})
	return resp
}
func (s *session) ReadInputRegisters(startingAddress uint16, length uint16) string {
	var resp string
	s.connection(func() {
		resp = s.client.ReadInputRegisters(startingAddress, length)
	})
	return resp
}
func (s *session) ReadHoldingRegisters(startingAddress uint16, length uint16) string {
	var resp string
	s.connection(func() {
		resp = s.client.ReadHoldingRegisters(startingAddress, length)
	})
	return resp
}

func (s *session) WriteCoils(startingAddress uint16, length uint16, hexStr string) {
	s.connection(func() {
		s.client.WriteCoils(startingAddress, length, hexStr)
	})
}

func (s *session) WriteHoldingRegisters(startingAddress uint16, length uint16, hexStr string) {
	s.connection(func() {
		s.client.WriteHoldingRegisters(startingAddress, length, hexStr)
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
