package servers

import (
	"errors"
	"fmt"
	"go-iot/pkg/core"

	"github.com/beego/beego/v2/core/logs"
)

var m map[core.NetType]func() core.NetServer = make(map[core.NetType]func() core.NetServer)
var instances map[string]core.NetServer = make(map[string]core.NetServer)

func RegServer(f func() core.NetServer) {
	s := f()
	m[s.Type()] = f
	logs.Info("Register Server [%s]", s.Type())
}

func StartServer(conf core.NetworkConf) error {
	if _, ok := instances[conf.ProductId]; ok {
		return errors.New("network is runing")
	}
	t := core.NetType(conf.Type)
	if f, ok := m[t]; ok {
		_, err := core.NewCodec(conf)
		if err != nil {
			return err
		}
		s := f()
		err = s.Start(conf)
		if err != nil {
			return err
		}
		instances[conf.ProductId] = s
		return nil
	} else {
		return fmt.Errorf("unknow type %s", conf.Type)
	}
}

func GetServer(productId string) core.NetServer {
	s := instances[productId]
	return s
}

func StopServer(productId string) error {
	server := GetServer(productId)
	if server == nil {
		return errors.New("network is not runing")
	}
	server.Stop()
	delete(instances, productId)
	return nil
}
