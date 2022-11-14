package eventbus

import (
	"sync"
)

const (
	// /device/productId/deviceId/property
	DeviceMessageTopic string = "/device/%s/%s/property"
)

var b = newEventBus()

func newEventBus() *eventBus {
	return &eventBus{
		m:       map[string][]func(data interface{}){},
		matcher: *NewAntPathMatcher(),
	}
}

type eventBus struct {
	sync.Mutex
	m       map[string][]func(data interface{})
	matcher AntPathMatcher
}

func (b *eventBus) match(pattern string, path string) bool {
	return b.matcher.Match(pattern, path)
}

func Subscribe(pattern string, run func(data interface{})) {
	b.Lock()
	defer b.Unlock()
	if _, ok := b.m[pattern]; ok {
		b.m[pattern] = append(b.m[pattern], run)
	}
}

func Publish(topic string, data interface{}) {
	b.Lock()
	defer b.Unlock()
	for pattern, listener := range b.m {
		if b.match(pattern, topic) {
			for _, callback := range listener {
				callback(data)
			}
		}
	}
}
