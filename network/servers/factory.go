package servers

import (
	"fmt"
	"go-iot/codec"
)

var m map[codec.NetServerType]func() codec.NetworkServer = make(map[codec.NetServerType]func() codec.NetworkServer)
var ins map[string]codec.NetworkServer = make(map[string]codec.NetworkServer)

func RegServer(f func() codec.NetworkServer) {
	s := f()
	m[s.Type()] = f
}

func StartServer(conf codec.NetworkConf) error {
	t := codec.NetServerType(conf.Type)
	if f, ok := m[t]; ok {
		s := f()
		s.Start(conf)
		ins[conf.ProductId] = s
		return nil
	} else {
		return fmt.Errorf("unknow type %s", conf.Type)
	}
}

func GetServer(productId string) codec.NetworkServer {
	s := ins[productId]
	return s
}
