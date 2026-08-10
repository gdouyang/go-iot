package eventpush

import (
	"container/list"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"go-iot/pkg/eventbus"

	"github.com/gorilla/websocket"
)

// 死连接（写失败）必须由自己的写 goroutine 移除，且不影响健康订阅者收数据
func TestDeadSubscriberIsolated(t *testing.T) {
	up := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	serverConns := make(chan *websocket.Conn, 2)
	done := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, _ := up.Upgrade(w, r, nil)
		serverConns <- ws
		<-done
	}))
	defer srv.Close()
	defer close(done)

	clientA, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientA.Close()
	connA := <-serverConns

	clientB, _, err := websocket.DefaultDialer.Dial("ws"+strings.TrimPrefix(srv.URL, "http"), nil)
	if err != nil {
		t.Fatal(err)
	}
	clientB.Close()
	connB := <-serverConns
	connB.Close() // 模拟死连接：服务端侧已断，写必然失败

	e := &eventHub{
		subscribe:   make(chan Subscriber, 10),
		unsubscribe: make(chan Subscriber, 10),
		publish:     make(chan eventbus.Message, 10),
		subscribers: list.New(),
		matcher:     eventbus.NewAntPathMatcher(),
	}
	go e.writeLoop()

	e.subscribe <- Subscriber{Topic: "/device/p/d/*", Addr: "alive", Conn: connA}
	e.subscribe <- Subscriber{Topic: "/device/p/d/*", Addr: "dead", Conn: connB}
	// subscribers 跨 goroutine 读需要锁（list 非线程安全）
	waitFor(t, func() bool {
		e.mu.Lock()
		defer e.mu.Unlock()
		return e.subscribers.Len() == 2
	})

	msg := eventbus.NewPropertiesMessage("d", "p", map[string]interface{}{"v": 1})
	e.publish <- &msg

	// 健康订阅者仍能收到
	_ = clientA.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, payload, err := clientA.ReadMessage()
	if err != nil || !strings.Contains(string(payload), `"deviceId":"d"`) {
		t.Fatalf("alive subscriber got no message: %v %s", err, payload)
	}
	// 死连接被其写 goroutine 移除
	waitFor(t, func() bool {
		e.mu.Lock()
		defer e.mu.Unlock()
		return e.subscribers.Len() == 1
	})
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met within 3s")
}
