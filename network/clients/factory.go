package clients

import (
	"fmt"
	"go-iot/codec"
)

var m map[codec.NetClientType]func() codec.NetworkClient = make(map[codec.NetClientType]func() codec.NetworkClient)

func RegClient(f func() codec.NetworkClient) {
	s := f()
	m[s.Type()] = f
}

func Connect(deviceId string, conf codec.NetworkConf) error {
	t := codec.NetClientType(conf.Type)
	if f, ok := m[t]; ok {
		s := f()
		return s.Connect(deviceId, conf)
	} else {
		return fmt.Errorf("unknow client type %s", conf.Type)
	}
}
