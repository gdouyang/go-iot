package tcpserver_test

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	_ "go-iot/pkg/codec"
	"go-iot/pkg/core"
	"go-iot/pkg/logger"
	"go-iot/pkg/network"
	tcpserver "go-iot/pkg/network/servers/tcp"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/store"
	"go-iot/pkg/timeseries"
	"go-iot/pkg/tsl"

	"github.com/stretchr/testify/require"
)

// ---- 短集成：上线 + 属性（校验 mock 时序真有数据）----

const scriptOnline = `
function OnConnect(context) {
  console.log("OnConnect")
}
function OnMessage(context) {
	var data = JSON.parse(context.MsgToString())
	context.DeviceOnline(data.deviceId)
	context.SaveProperties({"deviceId": data.deviceId, "temperature": 1.1})
}
function OnInvoke(context) {}
`

func TestTcpServer_DeviceOnlineAndSaveProperties(t *testing.T) {
	timeseries.MockReset()
	port := testhelper.FreeTCPPort(t)
	productId := fmt.Sprintf("tcp-p-%d", port)
	deviceId := "dev-tcp-1"
	testhelper.SetupProductDevice(t, productId, deviceId)

	conf := network.NetworkConf{
		Name:      "tcp-it",
		ProductId: productId,
		CodecId:   "script_codec",
		Port:      port,
		Script:    scriptOnline,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
			"delimeter":{"type":"Delimited","delimited":"}"}}`, port),
	}
	srv := tcpserver.NewServer()
	require.NoError(t, srv.Start(conf))
	t.Cleanup(func() { _ = srv.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)

	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	require.NoError(t, err)
	defer conn.Close()

	payload := fmt.Sprintf(`{"deviceId":"%s","data":"x"}`, deviceId)
	_, err = conn.Write([]byte(payload))
	require.NoError(t, err)
	time.Sleep(200 * time.Millisecond)

	require.NotNil(t, core.GetSession(deviceId), "device should be online after message")
	require.Equal(t, 1, timeseries.MockPropertiesCount())
	saved := timeseries.MockLastProperties()
	require.NotNil(t, saved)
	require.Equal(t, deviceId, fmt.Sprint(saved["deviceId"]))
	require.EqualValues(t, 1.1, saved["temperature"])
}

func TestTcpServer_InvalidConfig(t *testing.T) {
	srv := tcpserver.NewServer()
	err := srv.Start(network.NetworkConf{
		ProductId:     "tcp-bad",
		Port:          1,
		Configuration: `{bad json`,
	})
	require.Error(t, err)
}

func TestTcpServerSpec_FromJson(t *testing.T) {
	spec := &tcpserver.TcpServerSpec{}
	require.NoError(t, spec.FromJson(`{"host":"127.0.0.1","port":9,"delimeter":{"type":"Delimited","delimited":"\n"}}`))
	require.Equal(t, "127.0.0.1", spec.Host)
}

// ---- delimiter / SplitFunc：完整分包场景 + 断言 OnMessage/SaveProperties ----

// script1：JSON + } 分隔，上线并写属性（msg=整帧）
const script1 = `
function OnConnect(context) {
  console.log("OnConnect: " + JSON.stringify(context))
}
function OnMessage(context) {
	var raw = context.MsgToString()
	var data = JSON.parse(raw)
	console.log("OnMessage: deviceId = " + data.deviceId)
	context.DeviceOnline(data.deviceId)
	context.SaveProperties({"deviceId": data.deviceId, "temperature": 1.1, "msg": raw})
}
function OnInvoke(context) {
	console.log("OnInvoke: " + JSON.stringify(context))
}
`

// scriptRecord：任意分包内容原样 SaveProperties，用于 FixLength/SplitFunc 断言帧内容
const scriptRecord = `
function OnConnect(context) {}
function OnMessage(context) {
	var s = context.MsgToString()
	context.SaveProperties({deviceId: "1234", msg: s, temperature: 1})
}
function OnInvoke(context) {}
`

func setupLegacyProduct(t *testing.T) {
	t.Helper()
	logger.InitNop()
	timeseries.MockReset()
	core.RegDeviceStore(store.NewMockDeviceStore())
	product := &core.Product{
		Id:          "test-product",
		Config:      make(map[string]string),
		StorePolicy: "mock",
	}
	tslData := &tsl.TslData{}
	if err := tslData.FromJson(`{"properties":[{"id":"temperature","type":"float"},{"id":"msg","type":"string"}],"functions":[],"events":[]}`); err != nil {
		t.Fatal(err)
	}
	product.TslData = tslData
	require.NoError(t, core.PutProduct(product))
	require.NoError(t, core.PutDevice(core.NewDevice("1234", product.Id, 0)))
}

func newServer(t *testing.T, conf network.NetworkConf) *tcpserver.TcpServer {
	t.Helper()
	s := tcpserver.NewServer()
	require.NoError(t, s.Start(conf))
	t.Cleanup(func() { _ = s.Stop() })
	_, err := core.NewCodec(conf.CodecId, conf.ProductId, conf.Script)
	require.NoError(t, err)
	time.Sleep(50 * time.Millisecond)
	return s
}

// fixLengthPayload 长度固定 27，与 FixLength(27) 对齐。
func fixLengthPayload() string {
	// "aasss " + "2006-01-02 15:04:05" + "_\n" = 6+19+2 = 27
	return fmt.Sprintf("aasss %s_\n", time.Now().Format("2006-01-02 15:04:05"))
}

func TestServerDelimited(t *testing.T) {
	setupLegacyProduct(t)
	port := testhelper.FreeTCPPort(t)
	conf := network.NetworkConf{
		Name:      "test server",
		ProductId: "test-product",
		CodecId:   "script_codec",
		Port:      port,
		Script:    script1,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
			"delimeter":{"type":"Delimited","delimited":"}"}}`, port),
	}
	newServer(t, conf)
	const n = 5
	newClient1(t, conf, n, func(i int) string {
		return fmt.Sprintf(`{"deviceId": "1234", "data": "%d"}`, i)
	})

	// 5 帧 JSON 应以 } 分包，触发 5 次 SaveProperties
	require.Equal(t, n, timeseries.MockPropertiesCount(), "each delimited frame should SaveProperties once")
	require.NotNil(t, core.GetSession("1234"))
	last := timeseries.MockLastProperties()
	require.NotNil(t, last)
	require.Equal(t, "1234", fmt.Sprint(last["deviceId"]))
	require.EqualValues(t, 1.1, last["temperature"])
	require.Contains(t, fmt.Sprint(last["msg"]), `"data": "4"`) // 最后一帧 i=4
	// 所有帧 msg 均应以 } 结尾（分隔符读入）
	for _, p := range timeseries.MockAllProperties() {
		require.True(t, strings.HasSuffix(fmt.Sprint(p["msg"]), "}"), "frame=%q", p["msg"])
	}
}

func TestServerFixLenght(t *testing.T) {
	setupLegacyProduct(t)
	port := testhelper.FreeTCPPort(t)
	conf := network.NetworkConf{
		Name:      "test server",
		ProductId: "test-product",
		CodecId:   "script_codec",
		Port:      port,
		Script:    scriptRecord,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
			"delimeter":{"type":"FixLength","length":27}}`, port),
	}
	newServer(t, conf)
	const n = 5
	newClient1(t, conf, n, func(i int) string {
		p := fixLengthPayload()
		require.Equal(t, 27, len(p), "payload must match FixLength")
		return p
	})

	require.Equal(t, n, timeseries.MockPropertiesCount(), "each 27-byte frame should trigger OnMessage once")
	for _, p := range timeseries.MockAllProperties() {
		msg := fmt.Sprint(p["msg"])
		require.Len(t, msg, 27, "saved frame length")
		require.True(t, strings.HasPrefix(msg, "aasss "), "frame=%q", msg)
	}
}

func TestServerSplitFunc(t *testing.T) {
	// SplitFunc: Delimited("\n") → 按换行分包
	setupLegacyProduct(t)
	port := testhelper.FreeTCPPort(t)
	conf := network.NetworkConf{
		Name:      "test server",
		ProductId: "test-product",
		CodecId:   "script_codec",
		Port:      port,
		Script:    scriptRecord,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
	"delimeter": {
		"type":"SplitFunc",
	  "splitFunc":"function splitFunc(parser) { parser.AddHandler(function(data) { parser.AppendResult(data); parser.Complete() }); parser.Delimited(\"\\n\") }"
	}
	}`, port),
	}
	newServer(t, conf)
	const n = 5
	newClient1(t, conf, n, func(i int) string {
		return fmt.Sprintf("line-%d\n", i)
	})

	require.Equal(t, n, timeseries.MockPropertiesCount(), "each newline-delimited frame once")
	all := timeseries.MockAllProperties()
	// 帧内容应包含 line-i（是否含 \n 取决于解析器是否保留分隔符）
	for i := 0; i < n; i++ {
		found := false
		for _, p := range all {
			if strings.Contains(fmt.Sprint(p["msg"]), fmt.Sprintf("line-%d", i)) {
				found = true
				break
			}
		}
		require.True(t, found, "missing frame for line-%d, all=%v", i, all)
	}
}

func TestServerSplitFunc1(t *testing.T) {
	// 先按空格再按换行：一帧写入可能产生多段 OnMessage
	setupLegacyProduct(t)
	port := testhelper.FreeTCPPort(t)
	conf := network.NetworkConf{
		Name:      "test server",
		ProductId: "test-product",
		CodecId:   "script_codec",
		Port:      port,
		Script:    scriptRecord,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
	"delimeter": {
		"type":"SplitFunc",
	  "splitFunc":"function splitFunc(parser) { parser.AddHandler(function(data) { parser.AddHandler(function(data){ parser.AppendResult(data); parser.Complete() });  parser.Delimited(\"\\n\") }); parser.Delimited(\" \") }"
	}
	}`, port),
	}
	newServer(t, conf)
	const n = 5
	newClient1(t, conf, n, func(i int) string {
		// "partA partB\n" → 至少拆出两段
		return fmt.Sprintf("A%d B%d\n", i, i)
	})

	// 空格+换行嵌套：首段 A* 常被外层 Delimited(" ") 消费，OnMessage 多投递 B* 段
	cnt := timeseries.MockPropertiesCount()
	require.Equal(t, n, cnt, "expect one trailing frame per write (B0..B4), all=%v", timeseries.MockAllProperties())
	joined := ""
	for _, p := range timeseries.MockAllProperties() {
		joined += fmt.Sprint(p["msg"]) + "|"
	}
	require.Contains(t, joined, "B0")
	require.Contains(t, joined, "B4")
}

func TestServerSplitFunc2(t *testing.T) {
	// Fixed(6)+Fixed(21)：与 "aasss "+date+"_\n" = 6+21 对齐
	setupLegacyProduct(t)
	port := testhelper.FreeTCPPort(t)
	conf := network.NetworkConf{
		Name:      "test server",
		ProductId: "test-product",
		CodecId:   "script_codec",
		Port:      port,
		Script:    scriptRecord,
		Configuration: fmt.Sprintf(`{"host":"127.0.0.1","port":%d,"useTLS":false,
	"delimeter": {
		"type":"SplitFunc",
	  "splitFunc":"function splitFunc(parser) { parser.AddHandler(function(data) { parser.AddHandler(function(data){ parser.AppendResult(data); parser.Complete() });  parser.Fixed(21) }); parser.Fixed(6) }"
	}
	}`, port),
	}
	newServer(t, conf)
	const n = 5
	newClient1(t, conf, n, func(i int) string {
		p := fixLengthPayload()
		require.Equal(t, 27, len(p))
		return p
	})

	// Fixed(6) 常被当作首段消费，AppendResult 可能只投递 Fixed(21) 段（时间+_\n）
	cnt := timeseries.MockPropertiesCount()
	all := timeseries.MockAllProperties()
	require.GreaterOrEqual(t, cnt, n, "fixed split should produce frames, got %d all=%v", cnt, all)

	joined := ""
	for _, p := range all {
		joined += fmt.Sprint(p["msg"])
	}
	// 21 字节段：形如 "2006-01-02 15:04:05_\n"
	require.Contains(t, joined, "_", "all=%v", all)
	require.Regexp(t, `\d{4}-\d{2}-\d{2}`, joined)
	// 每次写入至少对应一帧 21 字节后缀（解析器可能丢掉 6 字节前缀不回调）
	var suffixFrames int
	for _, p := range all {
		msg := fmt.Sprint(p["msg"])
		if strings.Contains(msg, ":") && strings.Contains(msg, "_") {
			suffixFrames++
		}
	}
	require.GreaterOrEqual(t, suffixFrames, n, "expect time suffix frames, all=%v", all)
}

func newClient1(t *testing.T, conf network.NetworkConf, rounds int, call func(i int) string) {
	t.Helper()
	spec := tcpserver.TcpServerSpec{}
	require.NoError(t, spec.FromJson(conf.Configuration))
	spec.Port = conf.Port
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", spec.Host, spec.Port))
	require.NoError(t, err)
	defer conn.Close()

	go func() {
		stdin := bufio.NewScanner(conn)
		for stdin.Scan() {
			fmt.Println("server> " + stdin.Text())
		}
	}()

	for i := 0; i < rounds; i++ {
		str := call(i)
		_, err := conn.Write([]byte(str))
		require.NoError(t, err)
		fmt.Println("client> " + str)
		// 给读循环一点时间处理完本帧（比原先 1s 略短仍稳）
		time.Sleep(200 * time.Millisecond)
	}
	// 最后再等一会儿，确保最后一帧入库
	time.Sleep(100 * time.Millisecond)
}
