package modbus

import (
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
)

func decodePoint(p Point, batch Batch, data []byte) (any, error) {
	start := pduAddr(p.Address, 0)
	if batch.Address > start {
		return nil, fmt.Errorf("point %s address before batch", p.Id)
	}
	off := start - batch.Address
	switch p.Table {
	case COILS, DISCRETES_INPUT:
		bitIndex := int(off)
		byteI := bitIndex / 8
		if byteI >= len(data) {
			return nil, fmt.Errorf("point %s coil data short", p.Id)
		}
		bit := (data[byteI] >> (uint(bitIndex) % 8)) & 1
		eng := float64(bit)*scaleOf(p) + p.Offset
		if strings.ToLower(p.DataType) == "bool" {
			return eng != 0, nil
		}
		return applyScale(p, float64(bit)), nil
	default:
		byteOff := int(off) * 2
		need := int(p.Quantity) * 2
		if p.Quantity == 0 {
			need = 2
		}
		if byteOff+need > len(data) {
			return nil, fmt.Errorf("point %s register data short", p.Id)
		}
		raw := data[byteOff : byteOff+need]
		return decodeRegisters(p, raw)
	}
}

func decodeRegisters(p Point, raw []byte) (any, error) {
	dt := strings.ToLower(p.DataType)
	order := strings.ToUpper(strings.TrimSpace(p.ByteOrder))
	switch dt {
	case "string":
		if !utf8.Valid(raw) {
			return nil, fmt.Errorf("point %s invalid utf-8", p.Id)
		}
		s := strings.TrimRight(string(raw), "\x00")
		return s, nil
	case "bool":
		v := binary.BigEndian.Uint16(reorder16(raw, order))
		return applyScale(p, float64(v)) != 0, nil
	case "uint16":
		b := reorder16(raw, order)
		return applyScale(p, float64(binary.BigEndian.Uint16(b))), nil
	case "int16":
		b := reorder16(raw, order)
		return applyScale(p, float64(int16(binary.BigEndian.Uint16(b)))), nil
	case "uint32":
		b := reorder32(raw, order)
		return applyScale(p, float64(binary.BigEndian.Uint32(b))), nil
	case "int32":
		b := reorder32(raw, order)
		return applyScale(p, float64(int32(binary.BigEndian.Uint32(b)))), nil
	case "float32":
		b := reorder32(raw, order)
		return applyScale(p, float64(math.Float32frombits(binary.BigEndian.Uint32(b)))), nil
	case "uint64":
		b := reorder64(raw, order)
		return applyScale(p, float64(binary.BigEndian.Uint64(b))), nil
	case "int64":
		b := reorder64(raw, order)
		return applyScale(p, float64(int64(binary.BigEndian.Uint64(b)))), nil
	case "float64":
		b := reorder64(raw, order)
		return applyScale(p, math.Float64frombits(binary.BigEndian.Uint64(b))), nil
	default:
		return nil, fmt.Errorf("unsupported dataType %s", p.DataType)
	}
}

func scaleOf(p Point) float64 {
	if p.Scale == 0 {
		return 1
	}
	return p.Scale
}

func applyScale(p Point, raw float64) float64 {
	return raw*scaleOf(p) + p.Offset
}

func reorder16(b []byte, order string) []byte {
	if len(b) < 2 {
		return b
	}
	out := []byte{b[0], b[1]}
	if order == "BA" {
		out[0], out[1] = b[1], b[0]
	}
	return out
}

func reorder32(b []byte, order string) []byte {
	if len(b) < 4 {
		return b
	}
	src := [4]byte{b[0], b[1], b[2], b[3]}
	var out [4]byte
	switch order {
	case "CDAB":
		out = [4]byte{src[2], src[3], src[0], src[1]}
	case "BADC":
		out = [4]byte{src[1], src[0], src[3], src[2]}
	case "DCBA":
		out = [4]byte{src[3], src[2], src[1], src[0]}
	default: // ABCD or empty
		out = src
	}
	return out[:]
}

// reorder64: two 32-bit halves each apply the same 32-bit permutation.
func reorder64(b []byte, order string) []byte {
	if len(b) < 8 {
		return b
	}
	lo := reorder32(b[0:4], order)
	hi := reorder32(b[4:8], order)
	out := make([]byte, 8)
	copy(out[0:4], lo)
	copy(out[4:8], hi)
	return out
}

func valueChanged(oldV, newV any, deadband float64, numeric bool) bool {
	if oldV == nil {
		return true
	}
	if numeric {
		of, ok1 := toFloat(oldV)
		nf, ok2 := toFloat(newV)
		if !ok1 || !ok2 {
			return true
		}
		d := nf - of
		if d < 0 {
			d = -d
		}
		return d > deadband
	}
	return fmt.Sprint(oldV) != fmt.Sprint(newV)
}

func toFloat(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int16:
		return float64(t), true
	case int32:
		return float64(t), true
	case int64:
		return float64(t), true
	case uint16:
		return float64(t), true
	case uint32:
		return float64(t), true
	case uint64:
		return float64(t), true
	case bool:
		if t {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

func isNumericType(dataType string) bool {
	switch strings.ToLower(dataType) {
	case "bool", "string":
		return false
	default:
		return true
	}
}
