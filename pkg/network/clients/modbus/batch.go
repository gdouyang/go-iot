package modbus

import "sort"

// Batch is one Modbus PDU covering contiguous (or gapped) points of the same table.
type Batch struct {
	Table   string
	Address uint16
	Quantity uint16
	Points  []Point
}

// BuildBatches groups by table, sorts by PDU address, then greedily merges.
// cap = min(groupMax, tableLimit). gap is in register/bit units.
func BuildBatches(points []Point, addressBase int, groupMax uint16, gap uint16) []Batch {
	byTable := map[string][]Point{}
	for _, p := range points {
		byTable[p.Table] = append(byTable[p.Table], p)
	}
	tables := make([]string, 0, len(byTable))
	for t := range byTable {
		tables = append(tables, t)
	}
	sort.Strings(tables)

	var out []Batch
	for _, table := range tables {
		list := byTable[table]
		sort.Slice(list, func(i, j int) bool {
			ai := pduAddr(list[i].Address, addressBase)
			aj := pduAddr(list[j].Address, addressBase)
			if ai == aj {
				return list[i].Id < list[j].Id
			}
			return ai < aj
		})
		capN := tableLimit(table)
		if groupMax > 0 && groupMax < capN {
			capN = groupMax
		}
		var cur *Batch
		for _, p := range list {
			start := pduAddr(p.Address, addressBase)
			qty := p.Quantity
			if qty == 0 {
				qty = 1
			}
			if cur == nil {
				b := Batch{Table: table, Address: start, Quantity: qty, Points: []Point{p}}
				cur = &b
				continue
			}
			end := cur.Address + cur.Quantity
			mergedQty := (start + qty) - cur.Address
			if start <= end+gap && mergedQty <= capN && start+qty >= cur.Address {
				if start+qty > end {
					cur.Quantity = mergedQty
				}
				cur.Points = append(cur.Points, p)
				continue
			}
			out = append(out, *cur)
			b := Batch{Table: table, Address: start, Quantity: qty, Points: []Point{p}}
			cur = &b
		}
		if cur != nil {
			out = append(out, *cur)
		}
	}
	return out
}

func pduAddr(addr uint16, addressBase int) uint16 {
	if addressBase == 1 && addr > 0 {
		return addr - 1
	}
	return addr
}
