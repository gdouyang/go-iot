package util

import (
	"sync"
	"time"
)

var uuid int = int(time.Now().Unix())

var uuidLock sync.Mutex

// 获取UUID
func Uuid() int {
	uuidLock.Lock()
	defer uuidLock.Unlock()
	uuid += 1
	return uuid
}
