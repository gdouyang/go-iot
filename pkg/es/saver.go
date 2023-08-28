package es

import (
	"bytes"
	"sync"
	"time"

	logs "go-iot/pkg/logger"

	"github.com/elastic/go-elasticsearch/v8/esapi"
)

var DefaultEsSaveHelper EsDataSaveHelper = EsDataSaveHelper{
	dataCh:         make(chan string, DefaultEsConfig.BufferSize),
	lastCommitTime: time.Now().UnixMilli(),
}

type EsDataSaveHelper struct {
	sync.RWMutex
	batchData      []string
	dataCh         chan string
	batchTaskRun   bool
	lastCommitTime int64
}

// commit data to saver, every 5 sec send to bulk request to es
func Commit(index string, text string) {
	DefaultEsSaveHelper.commit(index, text)
}

func (t *EsDataSaveHelper) commit(index string, text string) {
	o := `{ "index" : { "_index" : "` + index + `" } }` + "\n" + text + "\n"
	t.dataCh <- o
	if len(t.dataCh) > (DefaultEsConfig.BufferSize / 2) {
		logs.Infof("commit data to es, chan length: %v", len(t.dataCh))
	}
	if !t.batchTaskRun {
		t.Lock()
		defer t.Unlock()
		if !t.batchTaskRun {
			t.batchTaskRun = true
			go t.batchSave()
		}
	}
}

func (t *EsDataSaveHelper) batchSave() {
	ticker := time.NewTicker(time.Millisecond * 5000)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: // 5秒内没有消息时保存
			t.save()
		case d := <-t.dataCh:
			t.batchData = append(t.batchData, d)
			if len(t.batchData) >= DefaultEsConfig.BulkSize {
				t.save()
			} else if t.lastCommitTime > 0 && time.Now().UnixMilli()-t.lastCommitTime >= 5000 { // 有消息但不够buff的
				t.lastCommitTime = time.Now().UnixMilli()
				t.save()
			}
		}
	}
}

func (t *EsDataSaveHelper) save() {
	if len(t.batchData) > 0 {
		var data []byte
		for i := 0; i < len(t.batchData); i++ {
			data = append(data, t.batchData[i]...)
		}
		// clear batch data
		t.batchData = t.batchData[:0]
		req := esapi.BulkRequest{
			Body: bytes.NewReader(data),
		}
		start := time.Now().UnixMilli()
		DoRequest(req)
		totalTime := time.Now().UnixMilli() - start
		if DefaultEsConfig.WarnTime > 0 && totalTime > int64(DefaultEsConfig.WarnTime) {
			logs.Warnf("save data to es use time: %v ms", totalTime)
		}
	}
}
