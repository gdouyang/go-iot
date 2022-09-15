package mqttproxy

import (
	"go-iot/models/network"

	"github.com/beego/beego/v2/core/logs"
)

var m = map[string]*Broker{}

func ServerStart(config string, script string) bool {
	spec := &network.MQTTProxySpec{}
	spec.FromJson(config)
	broker := NewBroker(spec, script)
	if broker == nil {
		logs.Error("broker %v start failed", spec.Name)
		return false
	} else {
		m[spec.Name] = broker
		return true
	}
}

func Meters(config string) map[string]int32 {
	spec := &network.MQTTProxySpec{}
	spec.FromJson(config)
	broker := m[spec.Name]
	if broker != nil {
		var rest = map[string]int32{}
		rest["TotalConnection"] = broker.TotalConnection()
		rest["TotalWasmVM"] = broker.TotalWasmVM()
		return rest
	}
	return nil
}
