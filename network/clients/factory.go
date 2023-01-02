package clients

import (
	"fmt"
	"go-iot/codec"

	"github.com/beego/beego/v2/core/logs"
)

var m map[codec.NetClientType]func() codec.NetClient = make(map[codec.NetClientType]func() codec.NetClient)
var instances map[string]codec.NetClient = make(map[string]codec.NetClient)

func RegClient(f func() codec.NetClient) {
	s := f()
	m[s.Type()] = f
	logs.Info("Client Register [%s]", s.Type())
}

func Connect(deviceId string, conf codec.NetworkConf) error {
	t := codec.NetClientType(conf.Type)
	if f, ok := m[t]; ok {
		_, err := codec.NewCodec(conf)
		if err != nil {
			return err
		}
		s := f()
		err = s.Connect(deviceId, conf)
		if err == nil {
			instances[deviceId] = s
		}
		return err
	} else {
		return fmt.Errorf("unknow client type %s", conf.Type)
	}
}

func GetClient(deviceId string) codec.NetClient {
	s := instances[deviceId]
	return s
}

func MqttMetaconfig() []codec.ProductMetaConfig {
	list := []codec.ProductMetaConfig{
		{Property: "host", Type: "string", Buildin: true, Desc: "The host of remote [127.0.0.1]"},
		{Property: "port", Type: "number", Buildin: true, Desc: "The port of remote"},
		{Property: "clientId", Type: "string", Buildin: true, Desc: ""},
		{Property: "username", Type: "string", Buildin: true, Desc: ""},
		{Property: "password", Type: "password", Buildin: true, Desc: ""},
	}
	return list
}

func TcpMetaconfig() []codec.ProductMetaConfig {
	list := []codec.ProductMetaConfig{
		{Property: "host", Type: "string", Buildin: true, Desc: "The host of remote [127.0.0.1]"},
		{Property: "port", Type: "number", Buildin: true, Desc: "The port of remote"},
	}
	return list
}

func ModbusMetaconfig() []codec.ProductMetaConfig {
	list := []codec.ProductMetaConfig{
		{Property: "address", Type: "string", Buildin: true, Desc: "The host of remote [127.0.0.1]"},
		{Property: "port", Type: "number", Buildin: true, Desc: "The port of remote"},
		{Property: "unitID", Type: "number", Buildin: true, Desc: ""},
		{Property: "timeout", Type: "number", Buildin: true, Value: "5", Desc: "Connect & Read timeout(seconds)"},
		{Property: "idleTimeout", Type: "number", Buildin: true, Value: "5", Desc: "Idle timeout(seconds) to close the connection"},

		// {Property: "baudRate", Type: "number", Buildin: true, Desc: ""},
		// {Property: "dataBits", Type: "number", Buildin: true, Desc: ""},
		// {Property: "stopBits", Type: "number", Buildin: true, Desc: ""},
		// {Property: "parity", Type: "number", Buildin: true, Desc: ""},
	}
	return list
}
