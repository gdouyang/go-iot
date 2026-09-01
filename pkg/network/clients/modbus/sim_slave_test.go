package modbus

import (
	"encoding/binary"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/eventbus"
	"go-iot/pkg/network/testhelper"
	"go-iot/pkg/timeseries"

	logs "go-iot/pkg/logger"

	"github.com/stretchr/testify/require"
)

type simRead struct {
	FC    byte
	Start uint16
	Qty   uint16
}

// simSlave 本机 TCP 模拟从站：真实收发 Modbus 报文，不是空实现。
type simSlave struct {
	ln      net.Listener
	mu      sync.Mutex
	holding map[uint16]uint16
	coils   map[uint16]bool
	reads   []simRead
	done    chan struct{}
	closeMu sync.Once
}

func startSimSlave(t *testing.T) *simSlave {
	t.Helper()
	logs.InitNop()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	s := &simSlave{
		ln:      ln,
		holding: map[uint16]uint16{},
		coils:   map[uint16]bool{},
		done:    make(chan struct{}),
	}
	go s.serve()
	t.Cleanup(s.Close)
	return s
}

func (s *simSlave) HostPort() (string, int) {
	a := s.ln.Addr().(*net.TCPAddr)
	return a.IP.String(), a.Port
}

func (s *simSlave) SetHolding(addr, val uint16) {
	s.mu.Lock()
	s.holding[addr] = val
	s.mu.Unlock()
}

func (s *simSlave) Holding(addr uint16) uint16 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.holding[addr]
}

func (s *simSlave) SetCoil(addr uint16, on bool) {
	s.mu.Lock()
	s.coils[addr] = on
	s.mu.Unlock()
}

func (s *simSlave) Reads() []simRead {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]simRead, len(s.reads))
	copy(out, s.reads)
	return out
}

func (s *simSlave) Close() {
	s.closeMu.Do(func() {
		close(s.done)
		_ = s.ln.Close()
	})
}

func (s *simSlave) serve() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				return
			}
		}
		go s.handle(conn)
	}
}

func (s *simSlave) handle(conn net.Conn) {
	defer conn.Close()
	for {
		_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
		hdr := make([]byte, 7)
		if _, err := io.ReadFull(conn, hdr); err != nil {
			return
		}
		length := binary.BigEndian.Uint16(hdr[4:6])
		if length < 2 || length > 260 {
			return
		}
		pdu := make([]byte, int(length)-1)
		if _, err := io.ReadFull(conn, pdu); err != nil {
			return
		}
		respPDU := s.dispatch(pdu)
		out := make([]byte, 7+len(respPDU))
		copy(out[0:4], hdr[0:4])
		binary.BigEndian.PutUint16(out[4:6], uint16(1+len(respPDU)))
		out[6] = hdr[6]
		copy(out[7:], respPDU)
		if _, err := conn.Write(out); err != nil {
			return
		}
	}
}

func (s *simSlave) dispatch(pdu []byte) []byte {
	if len(pdu) < 1 {
		return []byte{0x80, 0x01}
	}
	switch pdu[0] {
	case 0x01:
		return s.readBits(pdu, false)
	case 0x02:
		return s.readBits(pdu, false)
	case 0x03:
		return s.readRegs(pdu, true)
	case 0x04:
		return s.readRegs(pdu, true)
	case 0x05:
		return s.writeCoil(pdu)
	case 0x06:
		return s.writeSingleReg(pdu)
	case 0x0F:
		return s.writeCoils(pdu)
	case 0x10:
		return s.writeRegs(pdu)
	default:
		return []byte{pdu[0] | 0x80, 0x01}
	}
}

func (s *simSlave) readRegs(pdu []byte, holding bool) []byte {
	if len(pdu) < 5 {
		return []byte{pdu[0] | 0x80, 0x03}
	}
	start := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])
	if qty < 1 || qty > 125 {
		return []byte{pdu[0] | 0x80, 0x03}
	}
	data := make([]byte, int(qty)*2)
	s.mu.Lock()
	s.reads = append(s.reads, simRead{FC: pdu[0], Start: start, Qty: qty})
	for i := 0; i < int(qty); i++ {
		var v uint16
		if holding {
			v = s.holding[start+uint16(i)]
		}
		binary.BigEndian.PutUint16(data[i*2:], v)
	}
	s.mu.Unlock()
	out := make([]byte, 2+len(data))
	out[0] = pdu[0]
	out[1] = byte(len(data))
	copy(out[2:], data)
	return out
}

func (s *simSlave) readBits(pdu []byte, _ bool) []byte {
	if len(pdu) < 5 {
		return []byte{pdu[0] | 0x80, 0x03}
	}
	start := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])
	if qty < 1 || qty > 2000 {
		return []byte{pdu[0] | 0x80, 0x03}
	}
	nbytes := int((qty + 7) / 8)
	data := make([]byte, nbytes)
	s.mu.Lock()
	for i := 0; i < int(qty); i++ {
		if s.coils[start+uint16(i)] {
			data[i/8] |= 1 << (uint(i) % 8)
		}
	}
	s.mu.Unlock()
	out := make([]byte, 2+len(data))
	out[0] = pdu[0]
	out[1] = byte(len(data))
	copy(out[2:], data)
	return out
}

func (s *simSlave) writeSingleReg(pdu []byte) []byte {
	if len(pdu) < 5 {
		return []byte{0x86, 0x03}
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	val := binary.BigEndian.Uint16(pdu[3:5])
	s.mu.Lock()
	s.holding[addr] = val
	s.mu.Unlock()
	return pdu[:5]
}

func (s *simSlave) writeRegs(pdu []byte) []byte {
	if len(pdu) < 6 {
		return []byte{0x90, 0x03}
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])
	bc := int(pdu[5])
	if qty < 1 || len(pdu) < 6+bc {
		return []byte{0x90, 0x03}
	}
	s.mu.Lock()
	for i := 0; i < int(qty) && (6+i*2+1) < len(pdu); i++ {
		s.holding[addr+uint16(i)] = binary.BigEndian.Uint16(pdu[6+i*2 : 8+i*2])
	}
	s.mu.Unlock()
	out := make([]byte, 5)
	out[0] = 0x10
	binary.BigEndian.PutUint16(out[1:3], addr)
	binary.BigEndian.PutUint16(out[3:5], qty)
	return out
}

func (s *simSlave) writeCoil(pdu []byte) []byte {
	if len(pdu) < 5 {
		return []byte{0x85, 0x03}
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	val := binary.BigEndian.Uint16(pdu[3:5])
	s.mu.Lock()
	s.coils[addr] = val == 0xFF00
	s.mu.Unlock()
	return pdu[:5]
}

func (s *simSlave) writeCoils(pdu []byte) []byte {
	if len(pdu) < 6 {
		return []byte{0x8F, 0x03}
	}
	addr := binary.BigEndian.Uint16(pdu[1:3])
	qty := binary.BigEndian.Uint16(pdu[3:5])
	s.mu.Lock()
	for i := 0; i < int(qty); i++ {
		byteI := 6 + i/8
		if byteI >= len(pdu) {
			break
		}
		s.coils[addr+uint16(i)] = (pdu[byteI]>>uint(i%8))&1 == 1
	}
	s.mu.Unlock()
	out := make([]byte, 5)
	out[0] = 0x0F
	binary.BigEndian.PutUint16(out[1:3], addr)
	binary.BigEndian.PutUint16(out[3:5], qty)
	return out
}

func newTCPSession(t *testing.T, slave *simSlave) *ModbusSession {
	t.Helper()
	host, port := slave.HostPort()
	s := newSession()
	s.deviceId = "dev-mock"
	s.productId = "prod-mock"
	s.tcpInfo = &TcpInfo{
		Address:     host,
		Port:        port,
		UnitID:      1,
		Timeout:     2,
		IdleTimeout: 5,
	}
	t.Cleanup(func() { _ = s.Disconnect() })
	return s
}

func TestModbusClient_ReadHoldingRegistersFromSimSlave(t *testing.T) {
	slave := startSimSlave(t)
	slave.SetHolding(4, 105)
	host, port := slave.HostPort()
	cli, err := NewDeviceClient(ProtocolTCP, &TcpInfo{Address: host, Port: port, UnitID: 1, Timeout: 2, IdleTimeout: 5})
	require.NoError(t, err)
	require.NoError(t, cli.OpenConnection())
	t.Cleanup(func() { _ = cli.CloseConnection() })

	raw, err := cli.GetValue(HOLDING_REGISTERS, 4, 1)
	require.NoError(t, err)
	require.Equal(t, []byte{0x00, 0x69}, raw)
}

func TestModbusSession_ReadHoldingRegistersResponse(t *testing.T) {
	slave := startSimSlave(t)
	slave.SetHolding(4, 105)
	s := newTCPSession(t, slave)
	resp, err := s.ReadHoldingRegisters(4, 1)
	require.NoError(t, err)
	require.Equal(t, int16(105), resp.MsgToInt16())
	require.Equal(t, uint16(105), resp.MsgToUint16())
	require.Equal(t, "0069", resp.MsgToHexStr())
}

func TestModbusSession_WriteThenReadHolding(t *testing.T) {
	slave := startSimSlave(t)
	s := newTCPSession(t, slave)
	require.NoError(t, s.WriteHoldingRegisters(4, 1, "00c8"))
	resp, err := s.ReadHoldingRegisters(4, 1)
	require.NoError(t, err)
	require.Equal(t, uint16(200), resp.MsgToUint16())
	require.Equal(t, uint16(200), slave.Holding(4))
}

func TestModbusClient_ReadCoilsFromSimSlave(t *testing.T) {
	slave := startSimSlave(t)
	slave.SetCoil(1, true)
	host, port := slave.HostPort()
	cli, err := NewDeviceClient(ProtocolTCP, &TcpInfo{Address: host, Port: port, UnitID: 1, Timeout: 2, IdleTimeout: 5})
	require.NoError(t, err)
	require.NoError(t, cli.OpenConnection())
	t.Cleanup(func() { _ = cli.CloseConnection() })

	raw, err := cli.GetValue(COILS, 0, 8)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(raw), 1)
	require.Equal(t, byte(0x02), raw[0])
}

func TestCollectNow_FromSimSlaveSavesProperties(t *testing.T) {
	slave := startSimSlave(t)
	slave.SetHolding(4, 105)

	testhelper.InitCore()
	timeseries.MockReset()
	productId := "p-mock-collect"
	deviceId := "d-mock-collect"
	core.DeleteProduct(productId)
	core.DeleteDevice(deviceId)
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK,
		`{"properties":[{"id":"Temperature","name":"Temperature","type":"float"}]}`)
	require.NoError(t, err)
	product.NetworkType = "MODBUS"
	require.NoError(t, core.PutProduct(product))
	require.NoError(t, core.PutDevice(core.NewDevice(deviceId, productId, 0)))
	t.Cleanup(func() {
		DeleteCollector(productId)
		core.DeleteProduct(productId)
		core.DeleteDevice(deviceId)
	})

	PutCollector(productId, &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", IntervalMs: 1000, Report: ReportOnPeriod}},
		Points: []Point{{
			Id: "p-temp", PropertyId: "Temperature", GroupId: "g1",
			Table: HOLDING_REGISTERS, Address: 4, Quantity: 1,
			DataType: "int16", Scale: 0.1, ByteOrder: "AB", Access: AccessRead,
		}},
	})

	s := newTCPSession(t, slave)
	s.deviceId = deviceId
	s.productId = productId
	require.NoError(t, s.CollectNow("g1"))

	got := timeseries.MockLastProperties()
	require.NotNil(t, got)
	require.InDelta(t, 10.5, got["Temperature"], 1e-9)
	require.Equal(t, deviceId, got["deviceId"])
	require.Greater(t, s.lastCollect.Load(), int64(0))
}

func TestCollectNow_NonContiguousAddressesSplitFrames(t *testing.T) {
	slave := startSimSlave(t)
	slave.SetHolding(4, 105)
	slave.SetHolding(10, 200)

	testhelper.InitCore()
	timeseries.MockReset()
	productId := "p-mock-split"
	deviceId := "d-mock-split"
	core.DeleteProduct(productId)
	core.DeleteDevice(deviceId)
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK,
		`{"properties":[{"id":"Temperature","type":"float"},{"id":"Pressure","type":"float"}]}`)
	require.NoError(t, err)
	product.NetworkType = "MODBUS"
	require.NoError(t, core.PutProduct(product))
	require.NoError(t, core.PutDevice(core.NewDevice(deviceId, productId, 0)))
	t.Cleanup(func() {
		DeleteCollector(productId)
		core.DeleteProduct(productId)
		core.DeleteDevice(deviceId)
	})

	PutCollector(productId, &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", IntervalMs: 1000, Report: ReportOnPeriod, GapTolerance: 0}},
		Points: []Point{
			{Id: "p-temp", PropertyId: "Temperature", GroupId: "g1", Table: HOLDING_REGISTERS, Address: 4, Quantity: 1, DataType: "int16", Scale: 0.1, ByteOrder: "AB", Access: AccessRead},
			{Id: "p-pres", PropertyId: "Pressure", GroupId: "g1", Table: HOLDING_REGISTERS, Address: 10, Quantity: 1, DataType: "int16", Scale: 0.1, ByteOrder: "AB", Access: AccessRead},
		},
	})

	s := newTCPSession(t, slave)
	s.deviceId = deviceId
	s.productId = productId
	require.NoError(t, s.CollectNow("g1"))

	require.Equal(t, []simRead{
		{FC: 0x03, Start: 4, Qty: 1},
		{FC: 0x03, Start: 10, Qty: 1},
	}, slave.Reads())
	got := timeseries.MockLastProperties()
	require.NotNil(t, got)
	require.InDelta(t, 10.5, got["Temperature"], 1e-9)
	require.InDelta(t, 20.0, got["Pressure"], 1e-9)
}

func TestCollectNow_GapToleranceMergesNonContiguousFrames(t *testing.T) {
	slave := startSimSlave(t)
	slave.SetHolding(4, 105)
	slave.SetHolding(10, 200)

	testhelper.InitCore()
	timeseries.MockReset()
	productId := "p-mock-merge"
	deviceId := "d-mock-merge"
	core.DeleteProduct(productId)
	core.DeleteDevice(deviceId)
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK,
		`{"properties":[{"id":"Temperature","type":"float"},{"id":"Pressure","type":"float"}]}`)
	require.NoError(t, err)
	product.NetworkType = "MODBUS"
	require.NoError(t, core.PutProduct(product))
	require.NoError(t, core.PutDevice(core.NewDevice(deviceId, productId, 0)))
	t.Cleanup(func() {
		DeleteCollector(productId)
		core.DeleteProduct(productId)
		core.DeleteDevice(deviceId)
	})

	PutCollector(productId, &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", IntervalMs: 1000, Report: ReportOnPeriod, GapTolerance: 8, MaxQuantity: 125}},
		Points: []Point{
			{Id: "p-temp", PropertyId: "Temperature", GroupId: "g1", Table: HOLDING_REGISTERS, Address: 4, Quantity: 1, DataType: "int16", Scale: 0.1, ByteOrder: "AB", Access: AccessRead},
			{Id: "p-pres", PropertyId: "Pressure", GroupId: "g1", Table: HOLDING_REGISTERS, Address: 10, Quantity: 1, DataType: "int16", Scale: 0.1, ByteOrder: "AB", Access: AccessRead},
		},
	})

	s := newTCPSession(t, slave)
	s.deviceId = deviceId
	s.productId = productId
	require.NoError(t, s.CollectNow("g1"))

	require.Equal(t, []simRead{
		{FC: 0x03, Start: 4, Qty: 7},
	}, slave.Reads())
	got := timeseries.MockLastProperties()
	require.NotNil(t, got)
	require.InDelta(t, 10.5, got["Temperature"], 1e-9)
	require.InDelta(t, 20.0, got["Pressure"], 1e-9)
}

func TestCollectNow_ConnectFailPublishesDebugLog(t *testing.T) {
	logs.InitNop()
	testhelper.InitCore()
	productId := "p-mock-debug-fail"
	deviceId := "d-mock-debug-fail"
	core.DeleteProduct(productId)
	core.DeleteDevice(deviceId)
	product, err := core.NewProduct(productId, map[string]string{}, core.TIME_SERISE_MOCK,
		`{"properties":[{"id":"Temperature","type":"float"}]}`)
	require.NoError(t, err)
	product.NetworkType = "MODBUS"
	require.NoError(t, core.PutProduct(product))
	require.NoError(t, core.PutDevice(core.NewDevice(deviceId, productId, 0)))
	t.Cleanup(func() {
		DeleteCollector(productId)
		core.DeleteProduct(productId)
		core.DeleteDevice(deviceId)
	})
	PutCollector(productId, &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", IntervalMs: 1000, Report: ReportOnPeriod}},
		Points: []Point{{
			Id: "p-temp", PropertyId: "Temperature", GroupId: "g1",
			Table: HOLDING_REGISTERS, Address: 4, Quantity: 1,
			DataType: "int16", Access: AccessRead,
		}},
	})

	got := make(chan *eventbus.DebugMessage, 4)
	pattern := eventbus.GetDebugTopic(productId, deviceId)
	handler := func(msg eventbus.Message) {
		if d, ok := msg.(*eventbus.DebugMessage); ok {
			select {
			case got <- d:
			default:
			}
		}
	}
	eventbus.Subscribe(pattern, handler)
	t.Cleanup(func() { eventbus.UnSubscribe(pattern, handler) })

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	port := ln.Addr().(*net.TCPAddr).Port
	require.NoError(t, ln.Close())

	s := newSession()
	s.deviceId = deviceId
	s.productId = productId
	s.tcpInfo = &TcpInfo{Address: "127.0.0.1", Port: port, UnitID: 1, Timeout: 1, IdleTimeout: 1}
	t.Cleanup(func() { _ = s.Disconnect() })

	require.NoError(t, s.CollectNow("g1"))

	select {
	case msg := <-got:
		require.Equal(t, "error", msg.Level)
		require.Equal(t, deviceId, msg.DeviceId)
		require.Contains(t, strings.ToLower(msg.Data), "openconnection")
	case <-time.After(8 * time.Second):
		t.Fatal("timeout waiting collector debug log")
	}
}
