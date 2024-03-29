// 通知
package notify

import (
	"fmt"
	"sort"
	"sync"
)

var factory map[string]func() Notify = map[string]func() Notify{}
var instance map[int64]Notify = map[int64]Notify{}

type Notify interface {
	Kind() string
	Name() string
	Notify(message string) error
	FromJson(str NotifyConfig) error
	Meta() []map[string]string                        // 配置说明
	ParseTemplate(data map[string]interface{}) string // 消息模板
}

type NotifyConfig struct {
	Name     string
	Config   string
	Template string
}

func RegNotify(fn func() Notify) {
	notify := fn()
	factory[notify.Kind()] = fn
}

func GetAllNotify() []Notify {
	var all []Notify
	for _, value := range factory {
		all = append(all, value())
	}
	sort.Slice(all, func(i, j int) bool {
		return all[i].Kind() < all[j].Kind()
	})
	return all
}

func EnableNotify(kind string, id int64, config NotifyConfig) error {
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	if _, ok := instance[id]; ok {
		return fmt.Errorf("kind of %s notify is runing, id = %d", kind, id)
	}
	if fn, ok := factory[kind]; ok {
		notify := fn()
		err := notify.FromJson(config)
		if err != nil {
			return err
		}
		instance[id] = notify
		return nil
	}
	return fmt.Errorf("kind of %s notify not found", kind)
}

func DisableNotify(id int64) {
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	delete(instance, id)
}

func GetNotify(id int64) Notify {
	if notify, ok := instance[id]; ok {
		return notify
	}
	return nil
}

func TestNotify(kind string, config NotifyConfig) error {
	if fn, ok := factory[kind]; ok {
		n := fn()
		err := n.FromJson(config)
		if err != nil {
			return err
		}
		return n.Notify(n.ParseTemplate(map[string]interface{}{}))
	}
	return fmt.Errorf("kind of %s notify not found", kind)
}
