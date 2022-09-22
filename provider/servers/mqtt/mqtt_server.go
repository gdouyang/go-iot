package mqttserver

import (
	"go-iot/provider/codec"

	"github.com/beego/beego/v2/core/logs"
)

var m = map[string]*Broker{}

func ServerStart(network codec.Network) bool {
	spec := &MQTTServerSpec{}
	spec.FromJson(network.Configuration)
	broker := NewBroker(spec, network)
	if broker == nil {
		logs.Error("broker %v start failed", spec.Name)
		return false
	} else {
		m[spec.Name] = broker
		return true
	}
}

func Meters(config string) map[string]int32 {
	spec := &MQTTServerSpec{}
	spec.FromJson(config)
	broker := m[spec.Name]
	if broker != nil {
		var rest = map[string]int32{}
		rest["TotalConnection"] = broker.TotalConnection()
		return rest
	}
	return nil
}
