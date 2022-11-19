package notify

import (
	"fmt"
	"sync"

	"github.com/beego/beego/v2/core/logs"
)

var factory map[string]func() Notify = map[string]func() Notify{}
var instance map[int64]Notify = map[int64]Notify{}

type Notify interface {
	Kind() string
	Name() string
	Notify(title, message string) error
	FromJson(str string) error
	Config() []map[string]string
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
	return all
}

func EnableNotify(kind string, id int64, config string) error {
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

func DoSend(id int64, title, message string) {
	var mutex sync.Mutex
	mutex.Lock()
	defer mutex.Unlock()

	if notify, ok := instance[id]; ok {
		go func() {
			err := notify.Notify(title, message)
			if err != nil {
				logs.Error(err)
			}
		}()
	}
}
