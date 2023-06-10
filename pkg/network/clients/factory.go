package clients

import (
	"fmt"
	"go-iot/pkg/core"

	"github.com/beego/beego/v2/core/logs"
)

var m map[core.NetType]func() core.NetClient = make(map[core.NetType]func() core.NetClient)
var instances map[string]core.NetClient = make(map[string]core.NetClient)

func RegClient(f func() core.NetClient) {
	s := f()
	m[s.Type()] = f
	logs.Info("Register Client [%s]", s.Type())
}

func Connect(deviceId string, conf core.NetworkConf) error {
	t := core.NetType(conf.Type)
	if f, ok := m[t]; ok {
		_, err := core.NewCodec(conf)
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

func GetClient(deviceId string) core.NetClient {
	s := instances[deviceId]
	return s
}
