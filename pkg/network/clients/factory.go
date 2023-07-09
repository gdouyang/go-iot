package clients

import (
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"log"
	"sync"
)

var m sync.Map
var instances sync.Map

type CreaterFun func() network.NetClient

func RegClient(f CreaterFun) {
	s := f()
	m.Store(s.Type(), f)
	log.Printf("Register Client [%s]", s.Type())
}

func Connect(deviceId string, conf network.NetworkConf) error {
	t := network.NetType(conf.Type)
	if f, ok := m.Load(t); ok {
		_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
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

func GetClient(deviceId string) network.NetClient {
	s, ok := instances.Load(deviceId)
	if ok {
		return s.(network.NetClient)
	}
	return nil
}
