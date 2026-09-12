package eventbus

import (
	"reflect"
	"runtime/debug"
	"sync"

	logs "go-iot/pkg/logger"
)

// event bus of single node
type SingleNodeEventBus struct {
	sync.Mutex
	m       map[string][]func(data Message)
	matcher AntPathMatcher
}

func (bus *SingleNodeEventBus) match(pattern string, path string) bool {
	return bus.matcher.Match(pattern, path)
}

func (bus *SingleNodeEventBus) sub(pattern string, run func(msg Message)) {
	bus.Lock()
	defer bus.Unlock()
	bus.m[pattern] = append(bus.m[pattern], run)
}

func (bus *SingleNodeEventBus) unsub(pattern string, run func(data Message)) {
	bus.Lock()
	defer bus.Unlock()
	listener := bus.m[pattern]
	var l1 []func(data Message)
	removed := false
	sf2 := reflect.ValueOf(run)
	for _, callback := range listener {
		sf1 := reflect.ValueOf(callback)
		if !removed && sf1.Pointer() == sf2.Pointer() {
			removed = true
			continue
		}
		l1 = append(l1, callback)
	}
	if len(l1) == 0 {
		delete(bus.m, pattern)
	} else {
		bus.m[pattern] = l1
	}
}

func (bus *SingleNodeEventBus) publish(topic string, data Message) {
	bus.Lock()
	defer bus.Unlock()
	for pattern, listener := range bus.m {
		if bus.match(pattern, topic) {
			for _, callback := range listener {
				// 订阅者含用户规则、OTA 等外部逻辑：recover 兜底，
				// 单个订阅者 panic 不击穿整个进程（协议/HTTP 层已有同款保护）
				go func(cb func(data Message)) {
					defer func() {
						if r := recover(); r != nil {
							logs.Errorf("eventbus subscriber panic: %v\n%s", r, debug.Stack())
						}
					}()
					cb(data)
				}(callback)
			}
		}
	}
}
