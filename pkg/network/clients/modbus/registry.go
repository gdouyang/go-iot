package modbus

import (
	"encoding/json"
	"fmt"
	"sync"

	"go-iot/pkg/core"
	"go-iot/pkg/network"
	"go-iot/pkg/tsl"
)

func init() {
	core.RegCollectorRuntime(string(network.MODBUS), modbusCollectorRuntime{})
}

type modbusCollectorRuntime struct{}

func (modbusCollectorRuntime) Reload(productId string) {
	reloadCollectorCache(productId)
	reloadLocalCollectors(productId)
}

func (modbusCollectorRuntime) ReloadDevice(deviceId string) {
	reloadLocalCollectorDevice(deviceId)
}

func (modbusCollectorRuntime) Invalidate(productId string) {
	DeleteCollector(productId)
}

func (modbusCollectorRuntime) CollectNow(deviceId, groupId string) error {
	s := sessionByDevice(deviceId)
	if s != nil {
		return s.CollectNow(groupId)
	}
	if core.GetSession(deviceId) == nil {
		return fmt.Errorf("设备未连接")
	}
	return fmt.Errorf("设备不是 Modbus 会话")
}

func (modbusCollectorRuntime) Normalize(raw []byte, tslData *tsl.TslData) ([]byte, error) {
	cfg, err := ParseCollectorJSON(string(raw))
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(tslData); err != nil {
		return nil, err
	}
	if cfg.IsEmpty() {
		return nil, nil
	}
	return json.Marshal(cfg)
}

var liveSessions sync.Map // deviceId -> *ModbusSession

func registerSession(s *ModbusSession) {
	if s == nil || s.deviceId == "" {
		return
	}
	liveSessions.Store(s.deviceId, s)
}

func unregisterSession(deviceId string) {
	liveSessions.Delete(deviceId)
}

func reloadLocalCollectors(productId string) {
	liveSessions.Range(func(_, v any) bool {
		s, ok := v.(*ModbusSession)
		if !ok || s == nil {
			return true
		}
		if productId == "" || s.productId == productId {
			s.ReloadCollector()
		}
		return true
	})
}

func reloadLocalCollectorDevice(deviceId string) {
	if v, ok := liveSessions.Load(deviceId); ok {
		if s, ok := v.(*ModbusSession); ok && s != nil {
			s.ReloadCollector()
		}
	}
}

func sessionByDevice(deviceId string) *ModbusSession {
	if v, ok := liveSessions.Load(deviceId); ok {
		if s, ok := v.(*ModbusSession); ok {
			return s
		}
	}
	return nil
}
