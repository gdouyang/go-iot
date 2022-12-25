package modbus

import (
	"fmt"
	"go-iot/codec"
	"go-iot/network/clients"
	"sync"
	"time"

	"github.com/beego/beego/v2/core/logs"
)

func init() {
	clients.RegClient(func() codec.NetClient {
		return NewHodler()
	})
}

var once sync.Once
var driver *Hodler

type Hodler struct {
	mutex               sync.Mutex
	addressMap          map[string]chan bool
	workingAddressCount map[string]int
	stopped             bool
}

func NewHodler() *Hodler {
	once.Do(func() {
		driver = &Hodler{}
		driver.addressMap = make(map[string]chan bool)
		driver.workingAddressCount = make(map[string]int)
	})
	return driver
}

var concurrentCommandLimit = 100

func (c *Hodler) Type() codec.NetClientType {
	return codec.MODBUS_TCP
}
func (c *Hodler) Connect(deviceId string, network codec.NetworkConf) error {
	codec.PutSession(deviceId, &session{deviceId: deviceId})
	return nil
}

func (c *Hodler) Reload() error {
	return nil
}
func (c *Hodler) Close() error {
	c.stop(false)
	return nil
}

func (d *Hodler) stop(force bool) error {
	d.stopped = true
	if !force {
		d.waitAllCommandsToFinish()
	}
	for _, locked := range d.addressMap {
		close(locked)
	}
	return nil
}

// lockAddress mark address is unavailable because real device handle one request at a time
func (d *Hodler) lockAddress(address string) error {
	if d.stopped {
		return fmt.Errorf("service attempts to stop and unable to handle new request")
	}
	d.mutex.Lock()
	lock, ok := d.addressMap[address]
	if !ok {
		lock = make(chan bool, 1)
		d.addressMap[address] = lock
	}

	// workingAddressCount used to check high-frequency command execution to avoid goroutine block
	count, ok := d.workingAddressCount[address]
	if !ok {
		d.workingAddressCount[address] = 1
	} else if count >= concurrentCommandLimit {
		d.mutex.Unlock()
		errorMessage := fmt.Sprintf("High-frequency command execution. There are %v commands with the same address in the queue", concurrentCommandLimit)
		logs.Error(errorMessage)
		return fmt.Errorf(errorMessage)
	} else {
		d.workingAddressCount[address] = count + 1
	}

	d.mutex.Unlock()
	lock <- true

	return nil
}

// unlockAddress remove token after command finish
func (d *Hodler) unlockAddress(address string) {
	d.mutex.Lock()
	lock := d.addressMap[address]
	d.workingAddressCount[address] = d.workingAddressCount[address] - 1
	d.mutex.Unlock()
	<-lock
}

// lockableAddress return the lockable address according to the protocol
func (d *Hodler) lockableAddress(info *ConnectionInfo) string {
	var address string
	if info.Protocol == ProtocolTCP {
		address = fmt.Sprintf("%s:%d", info.Address, info.Port)
	} else {
		address = info.Address
	}
	return address
}

func (d *Hodler) HandleReadCommands(deviceName string, protocols map[string]ProtocolProperties, reqs []CommandRequest) (responses []*CommandValue, err error) {
	connectionInfo, err := createConnectionInfo(protocols)
	if err != nil {
		logs.Error("Fail to create read command connection info. err:%v \n", err)
		return responses, err
	}

	err = d.lockAddress(d.lockableAddress(connectionInfo))
	if err != nil {
		return responses, err
	}
	defer d.unlockAddress(d.lockableAddress(connectionInfo))

	responses = make([]*CommandValue, len(reqs))
	var deviceClient DeviceClient

	// create device client and open connection
	deviceClient, err = NewDeviceClient(connectionInfo)
	if err != nil {
		logs.Info("Read command NewDeviceClient failed. err:%v \n", err)
		return responses, err
	}

	err = deviceClient.OpenConnection()
	if err != nil {
		logs.Info("Read command OpenConnection failed. err:%v \n", err)
		return responses, err
	}

	defer func() { _ = deviceClient.CloseConnection() }()

	// handle command requests
	for i, req := range reqs {
		res, err := handleReadCommandRequest(deviceClient, req)
		if err != nil {
			logs.Info("Read command failed. Cmd:%v err:%v \n", "", err)
			return responses, err
		}

		responses[i] = res
	}

	return responses, nil
}

func handleReadCommandRequest(deviceClient DeviceClient, req CommandRequest) (*CommandValue, error) {
	var response []byte
	var result = &CommandValue{}
	var err error

	commandInfo, err := createCommandInfo(&req)
	if err != nil {
		return nil, err
	}

	response, err = deviceClient.GetValue(commandInfo)
	if err != nil {
		return result, err
	}

	result, err = TransformDataBytesToResult(&req, response, commandInfo)

	if err != nil {
		return result, err
	} else {
		logs.Info("Read command finished. Cmd:%v, %v \n", "req.DeviceResourceName", result)
	}

	return result, nil
}

func (d *Hodler) HandleWriteCommands(deviceName string, protocols map[string]ProtocolProperties, reqs []CommandRequest, params []*CommandValue) error {
	connectionInfo, err := createConnectionInfo(protocols)
	if err != nil {
		logs.Error("Fail to create write command connection info. err:%v \n", err)
		return err
	}

	err = d.lockAddress(d.lockableAddress(connectionInfo))
	if err != nil {
		return err
	}
	defer d.unlockAddress(d.lockableAddress(connectionInfo))

	var deviceClient DeviceClient

	// create device client and open connection
	deviceClient, err = NewDeviceClient(connectionInfo)
	if err != nil {
		return err
	}

	err = deviceClient.OpenConnection()
	if err != nil {
		return err
	}

	defer func() { _ = deviceClient.CloseConnection() }()

	// handle command requests
	for i, req := range reqs {
		err = handleWriteCommandRequest(deviceClient, req, params[i])
		if err != nil {
			logs.Error(err.Error())
			break
		}
	}

	return err
}

func handleWriteCommandRequest(deviceClient DeviceClient, req CommandRequest, param *CommandValue) error {
	var err error

	commandInfo, err := createCommandInfo(&req)
	if err != nil {
		return err
	}

	dataBytes, err := TransformCommandValueToDataBytes(commandInfo, param)
	if err != nil {
		return fmt.Errorf("transform command value failed, err: %v", err)
	}

	err = deviceClient.SetValue(commandInfo, dataBytes)
	if err != nil {
		return fmt.Errorf("handle write command request failed, err: %v", err)
	}

	logs.Info("Write command finished. Cmd:%v \n", "req.DeviceResourceName")
	return nil
}

// waitAllCommandsToFinish used to check and wait for the unfinished job
func (d *Hodler) waitAllCommandsToFinish() {
loop:
	for {
		for _, count := range d.workingAddressCount {
			if count != 0 {
				// wait a moment and check again
				time.Sleep(time.Second * SERVICE_STOP_WAIT_TIME)
				continue loop
			}
		}
		break loop
	}
}
