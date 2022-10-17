package clients

import (
	"fmt"
	"go-iot/codec"
)

var m map[codec.NetClientType]func() codec.NetworkClient = make(map[codec.NetClientType]func() codec.NetworkClient)
var instances map[string]codec.NetworkClient = make(map[string]codec.NetworkClient)

func RegClient(f func() codec.NetworkClient) {
	s := f()
	m[s.Type()] = f
}

func Connect(deviceId string, conf codec.NetworkConf) error {
	t := codec.NetClientType(conf.Type)
	if f, ok := m[t]; ok {
		s := f()
		err := s.Connect(deviceId, conf)
		if err == nil {
			instances[deviceId] = s
		}
		return err
	} else {
		return fmt.Errorf("unknow client type %s", conf.Type)
	}
}

func GetClient(deviceId string) codec.NetworkClient {
	s := instances[deviceId]
	return s
}
