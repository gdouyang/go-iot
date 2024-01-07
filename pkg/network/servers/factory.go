package servers

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"log"
	"sync"
)

type CreateFun func() network.NetServer

var m sync.Map
var instances sync.Map

func RegServer(f CreateFun) {
	s := f()
	m.Store(s.Type(), f)
	log.Printf("Register Server [%s]", s.Type())
}

func GetServer(productId string) network.NetServer {
	s, ok := instances.Load(productId)
	if ok {
		return s.(network.NetServer)
	}
	return nil
}

// 启动网络服务
func StartServer(conf network.NetworkConf) error {
	if _, ok := instances.Load(conf.ProductId); ok {
		return errors.New("network is runing")
	}
	t := network.NetType(conf.Type)
	if f, ok := m.Load(t); ok {
		_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
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

// 关闭网络服务
func StopServer(productId string) error {
	server := GetServer(productId)
	if server == nil {
		return errors.New("network is not runing")
	}
	server.Stop()
	instances.Delete(productId)
	return nil
}
