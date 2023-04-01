package eventbus

import (
	"reflect"
	"sync"
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
	for _, callback := range listener {
		sf1 := reflect.ValueOf(callback)
		sf2 := reflect.ValueOf(run)
		if sf1.Pointer() != sf2.Pointer() {
			l1 = append(l1, callback)
		}
	}
	bus.m[pattern] = l1
}

func (bus *SingleNodeEventBus) publish(topic string, data Message) {
	bus.Lock()
	defer bus.Unlock()
	for pattern, listener := range bus.m {
		if bus.match(pattern, topic) {
			for _, callback := range listener {
				go callback(data)
			}
		}
	}
}
