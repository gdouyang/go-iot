package modbus

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"

	"go-iot/pkg/tsl"
)

const (
	ReportOnPeriod         = "onPeriod"
	ReportOnChange         = "onChange"
	ReportOnChangeOrPeriod = "onChangeOrPeriod"

	AccessRead      = "read"
	AccessWrite     = "write"
	AccessReadWrite = "readwrite"

	maxPoints = 2000
	maxGroups = 20

	maxGroupIDLen   = 32
	maxPointIDLen   = 32
	maxGroupNameLen = 64

	holdingInputLimit = 125
	coilDiscreteLimit = 2000

	minIntervalMs = 100
	idPattern     = `^[0-9a-zA-Z_\-]+$`
)

var collectorIDRe = regexp.MustCompile(idPattern)

// CollectorConfig 产品级点表 + 采集组。
type CollectorConfig struct {
	Version              int     `json:"version"`
	Enabled              bool    `json:"enabled"`
	Mode                 string  `json:"mode"`
	AddressBase          int     `json:"addressBase"`
	OfflineAfterFailures int     `json:"offlineAfterFailures"`
	Groups               []Group `json:"groups"`
	Points               []Point `json:"points"`
}

type Group struct {
	Id               string  `json:"id"`
	Name             string  `json:"name,omitempty"`
	IntervalMs       int     `json:"intervalMs"`
	Report           string  `json:"report,omitempty"`
	MaxQuantity      uint16  `json:"maxQuantity,omitempty"`
	GapTolerance     uint16  `json:"gapTolerance,omitempty"`
	HeartbeatPeriods int     `json:"heartbeatPeriods,omitempty"`
	Deadband         float64 `json:"deadband,omitempty"`
}

type Point struct {
	Id         string  `json:"id"`
	PropertyId string  `json:"propertyId"`
	GroupId    string  `json:"groupId"`
	Table      string  `json:"table"`
	Address    uint16  `json:"address"`
	Quantity   uint16  `json:"quantity,omitempty"`
	DataType   string  `json:"dataType"`
	Scale      float64 `json:"scale,omitempty"`
	Offset     float64 `json:"offset,omitempty"`
	ByteOrder  string  `json:"byteOrder,omitempty"`
	Access     string  `json:"access,omitempty"`
}

func ParseCollectorJSON(raw string) (*CollectorConfig, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return &CollectorConfig{}, nil
	}
	var c CollectorConfig
	if err := json.Unmarshal([]byte(raw), &c); err != nil {
		return nil, fmt.Errorf("collector json: %w", err)
	}
	return &c, nil
}

func (c *CollectorConfig) IsEmpty() bool {
	if c == nil {
		return true
	}
	return len(c.Points) == 0 && len(c.Groups) == 0 && !c.Enabled
}

// ShouldStartCollector 产品启用且有组/点时启动采集。
func ShouldStartCollector(c *CollectorConfig) bool {
	if c == nil || !c.Enabled {
		return false
	}
	return len(c.Groups) > 0 && len(c.Points) > 0
}

func (c *CollectorConfig) Validate(tslData *tsl.TslData) error {
	if c == nil || c.IsEmpty() {
		return nil
	}
	if c.AddressBase != 0 && c.AddressBase != 1 {
		return fmt.Errorf("addressBase must be 0 or 1")
	}
	if c.OfflineAfterFailures < 0 {
		return fmt.Errorf("offlineAfterFailures must be >= 0")
	}
	if len(c.Groups) > maxGroups {
		return fmt.Errorf("groups must be <= %d", maxGroups)
	}
	if len(c.Points) > maxPoints {
		return fmt.Errorf("points must be <= %d", maxPoints)
	}
	if len(c.Points) > 0 && len(c.Groups) == 0 {
		return fmt.Errorf("points require at least one group")
	}

	groups := map[string]Group{}
	for i, g := range c.Groups {
		if !collectorIDRe.MatchString(g.Id) {
			return fmt.Errorf("groups[%d].id is invalid", i)
		}
		if len(g.Id) > maxGroupIDLen {
			return fmt.Errorf("groups[%d].id length must be <= %d", i, maxGroupIDLen)
		}
		if utf8.RuneCountInString(g.Name) > maxGroupNameLen {
			return fmt.Errorf("group [%s] name length must be <= %d", g.Id, maxGroupNameLen)
		}
		if _, ok := groups[g.Id]; ok {
			return fmt.Errorf("group id is repeat [%s]", g.Id)
		}
		if g.IntervalMs < minIntervalMs {
			return fmt.Errorf("group [%s] intervalMs must be >= %d", g.Id, minIntervalMs)
		}
		switch g.Report {
		case "", ReportOnPeriod, ReportOnChange, ReportOnChangeOrPeriod:
		default:
			return fmt.Errorf("group [%s] report is invalid", g.Id)
		}
		groups[g.Id] = g
	}

	propByKey := map[string]tsl.Property{}
	if tslData != nil {
		for _, p := range tslData.Properties {
			propByKey[strings.ToLower(strings.TrimSpace(p.GetId()))] = p
		}
	}

	seenPointID := map[string]bool{}
	seenProp := map[string]bool{}
	type span struct {
		start, end uint16
		idx        int
	}
	spans := map[string][]span{}

	for i := range c.Points {
		p := &c.Points[i]
		if !collectorIDRe.MatchString(p.Id) {
			return fmt.Errorf("points[%d].id is invalid", i)
		}
		if len(p.Id) > maxPointIDLen {
			return fmt.Errorf("points[%d].id length must be <= %d", i, maxPointIDLen)
		}
		if seenPointID[p.Id] {
			return fmt.Errorf("point id is repeat [%s]", p.Id)
		}
		seenPointID[p.Id] = true
		if _, ok := groups[p.GroupId]; !ok {
			return fmt.Errorf("point [%s] groupId [%s] not found", p.Id, p.GroupId)
		}
		switch p.Table {
		case HOLDING_REGISTERS, INPUT_REGISTERS, COILS, DISCRETES_INPUT:
		default:
			return fmt.Errorf("point [%s] table is invalid", p.Id)
		}
		if c.AddressBase == 1 && p.Address < 1 {
			return fmt.Errorf("point [%s] address must be >= 1 when addressBase=1", p.Id)
		}
		need, err := defaultQuantity(p.DataType)
		if err != nil {
			return fmt.Errorf("point [%s]: %w", p.Id, err)
		}
		if p.DataType == "string" {
			if p.Quantity < 1 {
				return fmt.Errorf("point [%s] string quantity must be >= 1", p.Id)
			}
			if p.Table == COILS || p.Table == DISCRETES_INPUT {
				return fmt.Errorf("point [%s] string is not allowed on coils/discretes", p.Id)
			}
		} else if p.Quantity == 0 {
			p.Quantity = need
		} else if p.Quantity < need {
			return fmt.Errorf("point [%s] quantity must be >= %d", p.Id, need)
		}
		if err := validateByteOrder(p.DataType, p.ByteOrder); err != nil {
			return fmt.Errorf("point [%s]: %w", p.Id, err)
		}
		switch p.Access {
		case "", AccessRead, AccessWrite, AccessReadWrite:
		default:
			return fmt.Errorf("point [%s] access is invalid", p.Id)
		}
		if p.Scale == 0 {
			p.Scale = 1
		}

		if tslData != nil {
			key := strings.ToLower(strings.TrimSpace(p.PropertyId))
			prop, ok := propByKey[key]
			if !ok {
				return fmt.Errorf("point [%s] propertyId [%s] not in tsl", p.Id, p.PropertyId)
			}
			if prop.GetType() == tsl.TypeObject {
				return fmt.Errorf("point [%s] cannot map to object property", p.Id)
			}
			canon := prop.GetId()
			if seenProp[strings.ToLower(canon)] {
				return fmt.Errorf("propertyId is repeat [%s]", canon)
			}
			seenProp[strings.ToLower(canon)] = true
			p.PropertyId = canon
		}

		pdu := p.Address
		if c.AddressBase == 1 {
			pdu = p.Address - 1
		}
		end := pdu + p.Quantity
		list := spans[p.Table]
		for _, s := range list {
			if pdu < s.end && s.start < end {
				return fmt.Errorf("point [%s] overlaps address range on table %s", p.Id, p.Table)
			}
		}
		spans[p.Table] = append(list, span{start: pdu, end: end, idx: i})
	}
	return nil
}

func defaultQuantity(dataType string) (uint16, error) {
	switch strings.ToLower(dataType) {
	case "bool", "uint16", "int16":
		return 1, nil
	case "uint32", "int32", "float32":
		return 2, nil
	case "uint64", "int64", "float64":
		return 4, nil
	case "string":
		return 0, nil
	default:
		return 0, fmt.Errorf("dataType %s is not supported", dataType)
	}
}

func validateByteOrder(dataType, order string) error {
	dt := strings.ToLower(dataType)
	if dt == "bool" || dt == "string" {
		return nil
	}
	order = strings.ToUpper(strings.TrimSpace(order))
	width := 16
	switch dt {
	case "uint16", "int16":
		width = 16
	case "uint32", "int32", "float32":
		width = 32
	case "uint64", "int64", "float64":
		width = 64
	}
	if width == 16 {
		if order == "" || order == "AB" || order == "BA" {
			return nil
		}
		return fmt.Errorf("16-bit byteOrder must be AB or BA")
	}
	switch order {
	case "", "ABCD", "CDAB", "BADC", "DCBA":
		return nil
	default:
		return fmt.Errorf("32/64-bit byteOrder must be ABCD, CDAB, BADC or DCBA")
	}
}

func (c *CollectorConfig) PDUAddress(addr uint16) uint16 {
	if c != nil && c.AddressBase == 1 && addr > 0 {
		return addr - 1
	}
	return addr
}

func tableLimit(table string) uint16 {
	switch table {
	case HOLDING_REGISTERS, INPUT_REGISTERS:
		return holdingInputLimit
	case COILS, DISCRETES_INPUT:
		return coilDiscreteLimit
	default:
		return holdingInputLimit
	}
}

func readPointsOfGroup(c *CollectorConfig, groupId string) []Point {
	if c == nil {
		return nil
	}
	var out []Point
	for _, p := range c.Points {
		acc := p.Access
		if acc == "" {
			acc = AccessRead
		}
		if p.GroupId == groupId && (acc == AccessRead || acc == AccessReadWrite) {
			out = append(out, p)
		}
	}
	return out
}
