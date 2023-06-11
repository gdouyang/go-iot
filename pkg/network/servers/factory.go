package servers

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

type CreateFun func() core.NetServer

var m sync.Map
var instances sync.Map

func RegServer(f CreateFun) {
	s := f()
	m.Store(s.Type(), f)
	logs.Info("Register Server [%s]", s.Type())
}

func StartServer(conf core.NetworkConf) error {
	if _, ok := instances.Load(conf.ProductId); ok {
		return errors.New("network is runing")
	}
	t := core.NetType(conf.Type)
	if f, ok := m.Load(t); ok {
		_, err := core.NewCodec(conf)
		if err != nil {
			return err
		}
		s := f.(CreateFun)()
		err = s.Start(conf)
		if err != nil {
			return err
		}
		instances.Store(conf.ProductId, s)
		return nil
	} else {
		return fmt.Errorf("unknow type %s", conf.Type)
	}
}

func GetServer(productId string) core.NetServer {
	s, ok := instances.Load(productId)
	if ok {
		return s.(core.NetServer)
	}
	return nil
}

func StopServer(productId string) error {
	server := GetServer(productId)
	if server == nil {
		return errors.New("network is not runing")
	}
	server.Stop()
	instances.Delete(productId)
	return nil
}
