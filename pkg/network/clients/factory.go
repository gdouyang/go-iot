package clients

import (
	"fmt"
	"go-iot/pkg/core"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

var m sync.Map
var instances sync.Map

type CreaterFun func() core.NetClient

func RegClient(f CreaterFun) {
	s := f()
	m.Store(s.Type(), f)
	logs.Info("Register Client [%s]", s.Type())
}

func Connect(deviceId string, conf core.NetworkConf) error {
	t := core.NetType(conf.Type)
	if f, ok := m.Load(t); ok {
		_, err := core.NewCodec(conf)
		if err != nil {
			return err
		}
		fun := f.(CreaterFun)
		s := fun()
		err = s.Connect(deviceId, conf)
		if err == nil {
			instances.Store(deviceId, s)
		}
		return err
	} else {
		return fmt.Errorf("unknow client type %s", conf.Type)
	}
}

func GetClient(deviceId string) core.NetClient {
	s, ok := instances.Load(deviceId)
	if ok {
		return s.(core.NetClient)
	}
	return nil
}
