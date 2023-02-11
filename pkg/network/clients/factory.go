package clients

import (
	"fmt"
	"go-iot/pkg/codec"

	"github.com/beego/beego/v2/core/logs"
)

var m map[codec.NetType]func() codec.NetClient = make(map[codec.NetType]func() codec.NetClient)
var instances map[string]codec.NetClient = make(map[string]codec.NetClient)

func RegClient(f func() codec.NetClient) {
	s := f()
	m[s.Type()] = f
	logs.Info("Client Register [%s]", s.Type())
}

func Connect(deviceId string, conf codec.NetworkConf) error {
	t := codec.NetType(conf.Type)
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
