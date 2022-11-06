package servers

import (
	"fmt"
	"go-iot/codec"

	"github.com/beego/beego/v2/core/logs"
)

var m map[codec.NetServerType]func() codec.NetServer = make(map[codec.NetServerType]func() codec.NetServer)
var instances map[string]codec.NetServer = make(map[string]codec.NetServer)

func RegServer(f func() codec.NetServer) {
	s := f()
	m[s.Type()] = f
	logs.Info("Server Register [%s]", s.Type())
}

func StartServer(conf codec.NetworkConf) error {
	t := codec.NetServerType(conf.Type)
	if f, ok := m[t]; ok {
		s := f()
		err := s.Start(conf)
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
