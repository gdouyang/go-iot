package modbus

import (
	"encoding/binary"
	"math"
	"testing"

	"go-iot/pkg/tsl"

	"github.com/stretchr/testify/require"
)

func TestValidateGroupIdAndNameMaxLength(t *testing.T) {
	td := tsl.NewTslData()
	require.NoError(t, td.FromJson(`{"properties":[{"id":"Temperature","name":"t","type":"float"}]}`))
	ok := &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", Name: "快扫", IntervalMs: 1000}},
		Points: []Point{{
			Id: "p1", PropertyId: "Temperature", GroupId: "g1",
			Table: HOLDING_REGISTERS, Address: 0, DataType: "int16",
		}},
	}
	require.NoError(t, ok.Validate(td))

	longID := &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "abcdefghijklmnopqrstuvwxyz0123456", IntervalMs: 1000}},
		Points:  []Point{{Id: "p1", PropertyId: "Temperature", GroupId: "abcdefghijklmnopqrstuvwxyz0123456", Table: HOLDING_REGISTERS, Address: 0, DataType: "int16"}},
	}
	require.Equal(t, 33, len(longID.Groups[0].Id))
	require.Error(t, longID.Validate(td))

	runes := make([]rune, 65)
	for i := range runes {
		runes[i] = '扫'
	}
	longName := &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", Name: string(runes), IntervalMs: 1000}},
		Points:  []Point{{Id: "p1", PropertyId: "Temperature", GroupId: "g1", Table: HOLDING_REGISTERS, Address: 0, DataType: "int16"}},
	}
	require.Error(t, longName.Validate(td))

	longPoint := &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", IntervalMs: 1000}},
		Points: []Point{{
			Id: "abcdefghijklmnopqrstuvwxyz0123456", PropertyId: "Temperature", GroupId: "g1",
			Table: HOLDING_REGISTERS, Address: 0, DataType: "int16",
		}},
	}
	require.Equal(t, 33, len(longPoint.Points[0].Id))
	require.Error(t, longPoint.Validate(td))
}

func TestValidateRejectsUnknownModeAndOverlap(t *testing.T) {
	td := tsl.NewTslData()
	require.NoError(t, td.FromJson(`{"properties":[{"id":"Temperature","name":"t","type":"float"}],"events":[],"functions":[]}`))

	c := &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "fast", IntervalMs: 1000}},
		Points: []Point{{
			Id: "p1", PropertyId: "Temperature", GroupId: "fast",
			Table: HOLDING_REGISTERS, Address: 0, DataType: "int16",
		}},
	}
	require.NoError(t, c.Validate(td))

	c.Points = append(c.Points, Point{
		Id: "p2", PropertyId: "Temperature", GroupId: "fast",
		Table: HOLDING_REGISTERS, Address: 0, DataType: "int16",
	})
	require.Error(t, c.Validate(td))
}

func TestValidatePropertyIdCanonicalAndObjectRejected(t *testing.T) {
	td := tsl.NewTslData()
	require.NoError(t, td.FromJson(`{"properties":[{"id":"Temperature","name":"t","type":"float"},{"id":"Nested","name":"n","type":"object","properties":[{"id":"a","name":"a","type":"int"}]}],"events":[],"functions":[]}`))

	c := &CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g1", IntervalMs: 1000}},
		Points: []Point{{
			Id: "p1", PropertyId: "temperature", GroupId: "g1",
			Table: HOLDING_REGISTERS, Address: 1, DataType: "int16", ByteOrder: "AB",
		}},
		AddressBase: 1,
	}
	require.NoError(t, c.Validate(td))
	require.Equal(t, "Temperature", c.Points[0].PropertyId)

	c.Points[0].PropertyId = "Nested"
	require.Error(t, c.Validate(td))
}

func TestBuildBatchesByTableAndCap(t *testing.T) {
	points := []Point{
		{Id: "a", Table: HOLDING_REGISTERS, Address: 0, Quantity: 1, DataType: "int16"},
		{Id: "b", Table: HOLDING_REGISTERS, Address: 1, Quantity: 1, DataType: "int16"},
		{Id: "c", Table: HOLDING_REGISTERS, Address: 10, Quantity: 1, DataType: "int16"},
		{Id: "d", Table: COILS, Address: 0, Quantity: 1, DataType: "bool"},
	}
	batches := BuildBatches(points, 0, 125, 0)
	require.Len(t, batches, 3)

	gapped := BuildBatches(points[:3], 0, 125, 8)
	require.Len(t, gapped, 1)
	require.Equal(t, uint16(11), gapped[0].Quantity)

	many := make([]Point, 0, 130)
	for i := 0; i < 130; i++ {
		many = append(many, Point{Id: string(rune('a'+i%26)) + string(rune('0'+i/26)), Table: HOLDING_REGISTERS, Address: uint16(i), Quantity: 1})
	}
	split := BuildBatches(many, 0, 2000, 0)
	require.GreaterOrEqual(t, len(split), 2)
	require.LessOrEqual(t, int(split[0].Quantity), 125)
}

func TestDecodeInt16ScaleAndCoilLSB(t *testing.T) {
	p := Point{Id: "t", PropertyId: "Temperature", Table: HOLDING_REGISTERS, Address: 0, Quantity: 1, DataType: "int16", Scale: 0.1, ByteOrder: "AB"}
	b := Batch{Table: HOLDING_REGISTERS, Address: 0, Quantity: 1, Points: []Point{p}}
	raw := []byte{0x00, 0x69} // 105
	v, err := decodePoint(p, b, raw)
	require.NoError(t, err)
	require.InDelta(t, 10.5, v, 1e-9)

	pBA := p
	pBA.ByteOrder = "BA"
	v2, err := decodePoint(pBA, b, []byte{0x69, 0x00})
	require.NoError(t, err)
	require.InDelta(t, 10.5, v2, 1e-9)

	coil := Point{Id: "c", Table: COILS, Address: 1, Quantity: 1, DataType: "bool"}
	cb := Batch{Table: COILS, Address: 0, Quantity: 8, Points: []Point{coil}}
	cv, err := decodePoint(coil, cb, []byte{0x02}) // bit 1
	require.NoError(t, err)
	require.Equal(t, true, cv)
}

func TestDecode64CDABTwoHalves(t *testing.T) {
	// 64-bit CDAB: each 32-bit half word-swapped.
	// Native ABCD bytes for uint64 0x0102030405060708: 01 02 03 04 05 06 07 08
	// CDAB halves: 03 04 01 02 | 07 08 05 06
	p := Point{Id: "x", Table: HOLDING_REGISTERS, Address: 0, Quantity: 4, DataType: "uint64", ByteOrder: "CDAB", Scale: 1}
	wire := []byte{0x03, 0x04, 0x01, 0x02, 0x07, 0x08, 0x05, 0x06}
	v, err := decodeRegisters(p, wire)
	require.NoError(t, err)
	require.Equal(t, float64(0x0102030405060708), v)

	p32 := Point{Id: "f", Table: HOLDING_REGISTERS, Address: 0, Quantity: 2, DataType: "float32", ByteOrder: "ABCD", Scale: 1}
	bits := math.Float32bits(12.5)
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, bits)
	fv, err := decodeRegisters(p32, buf)
	require.NoError(t, err)
	require.InDelta(t, 12.5, fv, 1e-5)
}

func TestParseCollectorEmpty(t *testing.T) {
	c, err := ParseCollectorJSON("")
	require.NoError(t, err)
	require.True(t, c.IsEmpty())
	require.False(t, ShouldStartCollector(c))
}

func TestCollectorRuntimeNormalize(t *testing.T) {
	rt := modbusCollectorRuntime{}
	td := tsl.NewTslData()
	require.NoError(t, td.FromJson(`{"properties":[{"id":"Temperature","name":"t","type":"float"}]}`))

	out, err := rt.Normalize([]byte(`{}`), td)
	require.NoError(t, err)
	require.Nil(t, out)

	body := `{"enabled":true,"groups":[{"id":"g1","intervalMs":1000}],"points":[{"id":"p1","propertyId":"Temperature","groupId":"g1","table":"HOLDING_REGISTERS","address":0,"dataType":"int16"}]}`
	out, err = rt.Normalize([]byte(body), td)
	require.NoError(t, err)
	require.Contains(t, string(out), `"g1"`)

	bad := `{"enabled":true,"groups":[{"id":"g1","intervalMs":1000}],"points":[{"id":"p1","propertyId":"Unknown","groupId":"g1","table":"HOLDING_REGISTERS","address":0,"dataType":"int16"}]}`
	_, err = rt.Normalize([]byte(bad), td)
	require.Error(t, err)
	require.Contains(t, err.Error(), "propertyId")
}

func TestShouldStartCollectorRequiresEnabledGroupsAndPoints(t *testing.T) {
	require.False(t, ShouldStartCollector(&CollectorConfig{Enabled: true}))
	require.False(t, ShouldStartCollector(&CollectorConfig{
		Enabled: true, Groups: []Group{{Id: "g", IntervalMs: 1000}},
	}))
	require.True(t, ShouldStartCollector(&CollectorConfig{
		Enabled: true,
		Groups:  []Group{{Id: "g", IntervalMs: 1000}},
		Points:  []Point{{Id: "p", PropertyId: "Temperature", GroupId: "g"}},
	}))
}

func TestNewCollectMessageCopiesProperties(t *testing.T) {
	src := map[string]any{"Temperature": 10.5}
	msg := newCollectMessage("fast", src)
	require.Equal(t, "collect", msg["source"])
	require.Equal(t, "fast", msg["groupId"])
	props := msg["properties"].(map[string]any)
	props["Temperature"] = 0
	require.Equal(t, 10.5, src["Temperature"])
}
