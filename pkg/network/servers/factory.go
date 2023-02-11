package servers

import (
	"errors"
	"fmt"
	"go-iot/pkg/codec"

	"github.com/beego/beego/v2/core/logs"
)

var m map[codec.NetType]func() codec.NetServer = make(map[codec.NetType]func() codec.NetServer)
var instances map[string]codec.NetServer = make(map[string]codec.NetServer)

func RegServer(f func() codec.NetServer) {
	s := f()
	m[s.Type()] = f
	logs.Info("Server Register [%s]", s.Type())
}

func StartServer(conf codec.NetworkConf) error {
	if _, ok := instances[conf.ProductId]; ok {
		return errors.New("network is runing")
	}
	t := codec.NetType(conf.Type)
	if f, ok := m[t]; ok {
		_, err := codec.NewCodec(conf)
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

func GetServer(productId string) codec.NetServer {
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
