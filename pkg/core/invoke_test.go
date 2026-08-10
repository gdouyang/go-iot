package core

import (
	"go-iot/pkg/tsl"
	"testing"
	"time"
)

// 挂起 codec：OnInvoke 不回复，模拟设备对命令无响应（触发命令超时）
type hangingCodec struct{}

func (hangingCodec) OnConnect(ctx MessageContext) error  { return nil }
func (hangingCodec) OnMessage(ctx MessageContext) error  { return nil }
func (hangingCodec) OnInvoke(ctx FuncInvokeContext) error { time.Sleep(30 * time.Second); return nil }
func (hangingCodec) OnClose(ctx MessageContext) error     { return nil }

// mock 时序存储
type mockTimeSeries struct{}

func (mockTimeSeries) Id() string                                  { return "mock" }
func (mockTimeSeries) PublishModel(p *Product, m tsl.TslData) error { return nil }
func (mockTimeSeries) QueryProperty(p *Product, r TimeDataSearchRequest) (map[string]interface{}, error) {
	return nil, nil
}
func (mockTimeSeries) QueryLogs(p *Product, r TimeDataSearchRequest) (map[string]interface{}, error) {
	return nil, nil
}
func (mockTimeSeries) QueryEvent(p *Product, e string, r TimeDataSearchRequest) (map[string]interface{}, error) {
	return nil, nil
}
func (mockTimeSeries) SaveProperties(p *Product, d map[string]interface{}) error { return nil }
func (mockTimeSeries) SaveEvents(p *Product, e string, d map[string]interface{}) error {
	return nil
}
func (mockTimeSeries) SaveLogs(p *Product, d LogData) error { return nil }
func (mockTimeSeries) Del(p *Product) error                 { return nil }

// 端到端：命令超时（设备不回复）后，设备迟到应答不得阻塞发送方 goroutine
func TestDoCmdInvokeLateReplyNotBlocking(t *testing.T) {
	defaultSessions = NewMemorySessionManager()
	codecMap.Store("p1", hangingCodec{})
	timeSeriseMap.Store("mock", mockTimeSeries{})
	defaultStore = &mockDeviceStore{p: &Product{Id: "p1", StorePolicy: "mock"}}
	_ = defaultStore.PutDevice(&Device{Id: "d1", ProductId: "p1"})
	PutSession("d1", &mockSession{info: map[string]any{}}, false)

	msg := FuncInvoke{DeviceId: "d1", FunctionId: OTA_UPDATE, Timeout: 1}
	err := DoCmdInvoke(msg)
	if err == nil {
		t.Fatal("命令应超时返回")
	}

	// 模拟设备迟到应答（消费方已退出）：发送方必须能完成，不得永久阻塞
	done := make(chan struct{})
	go func() {
		replyMap.reply("d1", &FuncInvokeReply{Success: true})
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("迟到应答发送阻塞（goroutine 泄漏）")
	}
}
