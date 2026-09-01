package modbus

import (
	stdctx "context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/tsl"

	logs "go-iot/pkg/logger"
)

type CollectorRuntime struct {
	cancel   stdctx.CancelFunc
	ctx      stdctx.Context
	inFlight sync.WaitGroup
}

func (s *ModbusSession) startSchedulers() {
	s.stopSchedulers()
	ctx, cancel := stdctx.WithCancel(stdctx.Background())
	s.schedMu.Lock()
	s.schedCancel = cancel
	s.schedCtx = ctx
	s.schedMu.Unlock()
	s.schedGen.Add(1)

	product := core.GetProduct(s.productId)
	cfg := GetCollector(s.productId)

	if ShouldStartCollector(cfg) && product != nil {
		rt := &CollectorRuntime{}
		rt.ctx, rt.cancel = stdctx.WithCancel(ctx)
		s.runtime = rt
		rt.Start(s, cfg, product)
	}
}

func (s *ModbusSession) stopSchedulers() {
	s.schedGen.Add(1)
	s.schedMu.Lock()
	cancel := s.schedCancel
	rt := s.runtime
	s.schedCancel = nil
	s.runtime = nil
	s.schedMu.Unlock()
	if cancel != nil {
		cancel()
	}
	if rt != nil && rt.cancel != nil {
		rt.cancel()
	}
	done := make(chan struct{})
	go func() {
		if rt != nil {
			rt.inFlight.Wait()
		}
		close(done)
	}()
	timeout := 10 * time.Second
	if s.tcpInfo != nil && s.tcpInfo.Timeout > 0 {
		timeout = time.Duration(s.tcpInfo.Timeout) * time.Second
		if timeout < 10*time.Second {
			timeout = 10 * time.Second
		}
	}
	select {
	case <-done:
	case <-time.After(timeout):
		logs.Warnf("modbus stopSchedulers timeout device=%s", s.deviceId)
	}
}

func (s *ModbusSession) ReloadCollector() {
	s.stopSchedulers()
	if s.stopped.Load() {
		return
	}
	s.startSchedulers()
}

func (rt *CollectorRuntime) Start(s *ModbusSession, cfg *CollectorConfig, product *core.Product) {
	for i := range cfg.Groups {
		g := cfg.Groups[i]
		go rt.groupLoop(s, cfg, product, g)
	}
}

func (rt *CollectorRuntime) groupLoop(s *ModbusSession, cfg *CollectorConfig, product *core.Product, g Group) {
	var inFlight atomic.Bool
	rt.collectGroup(s, cfg, product, g, &inFlight)
	interval := time.Duration(g.IntervalMs) * time.Millisecond
	if interval < time.Duration(minIntervalMs)*time.Millisecond {
		interval = time.Duration(minIntervalMs) * time.Millisecond
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			if rt.ctx.Err() != nil {
				return
			}
			if !inFlight.CompareAndSwap(false, true) {
				logs.Warnf("modbus collect skip-if-busy device=%s group=%s", s.deviceId, g.Id)
				continue
			}
			rt.collectGroup(s, cfg, product, g, &inFlight)
		case <-rt.ctx.Done():
			return
		case <-s.done:
			return
		}
	}
}

func (rt *CollectorRuntime) collectGroup(s *ModbusSession, cfg *CollectorConfig, product *core.Product, g Group, inFlight *atomic.Bool) {
	if inFlight != nil {
		defer inFlight.Store(false)
	}
	rt.inFlight.Add(1)
	defer rt.inFlight.Done()
	startedGen := s.schedGen.Load()
	points := readPointsOfGroup(cfg, g.Id)
	if len(points) == 0 {
		return
	}
	batches := BuildBatches(points, cfg.AddressBase, g.MaxQuantity, g.GapTolerance)
	decoded := map[string]any{}
	pointType := map[string]string{}
	anyOK := false
	err := s.withConn(func(cli *ModbusClient) error {
		var last error
		for _, b := range batches {
			data, e := cli.GetValue(b.Table, b.Address, b.Quantity)
			if e != nil {
				last = e
				logs.Warnf("modbus batch read fail device=%s table=%s addr=%d qty=%d err=%v", s.deviceId, b.Table, b.Address, b.Quantity, e)
				s.debugLog("error", "collector batch read fail group=%s table=%s addr=%d qty=%d: %v", g.Id, b.Table, b.Address, b.Quantity, e)
				continue
			}
			for _, p := range b.Points {
				v, de := decodePoint(p, b, data)
				if de != nil {
					logs.Warnf("modbus decode fail point=%s err=%v", p.Id, de)
					s.debugLog("error", "collector decode fail group=%s point=%s: %v", g.Id, p.Id, de)
					continue
				}
				decoded[p.PropertyId] = v
				pointType[p.PropertyId] = p.DataType
				anyOK = true
			}
		}
		return last
	})
	if err != nil && !anyOK {
		s.bumpGroupFail(g.Id, cfg)
		return
	}
	if anyOK {
		s.resetGroupFail(g.Id)
	} else {
		s.bumpGroupFail(g.Id, cfg)
		return
	}
	if startedGen != s.schedGen.Load() {
		return
	}
	if !ShouldStartCollector(cfg) {
		return
	}
	s.dispatchCollected(product, g, decoded, pointType)
}

func (s *ModbusSession) dispatchCollected(product *core.Product, g Group, decoded map[string]any, types map[string]string) {
	if !s.groupShouldReport(g, decoded, types) {
		return
	}
	cdc := core.GetCodec(s.productId)
	if cdc == nil {
		s.saveCollected(product, decoded)
		return
	}
	ctx := &collectContext{
		BaseContext: core.BaseContext{
			DeviceId:  s.deviceId,
			ProductId: s.productId,
			Session:   s,
		},
		msg: newCollectMessage(g.Id, decoded),
	}
	err := cdc.OnMessage(ctx)
	if err == nil {
		s.lastCollect.Store(time.Now().UnixMilli())
		return
	}
	if errors.Is(err, core.ErrFunctionNotImpl) {
		s.saveCollected(product, decoded)
		return
	}
	logs.Warnf("modbus OnMessage device=%s group=%s err=%v", s.deviceId, g.Id, err)
	s.debugLog("error", "collector OnMessage group=%s: %v", g.Id, err)
}

func (s *ModbusSession) saveCollected(product *core.Product, decoded map[string]any) {
	if product == nil || len(decoded) == 0 {
		return
	}
	data := make(map[string]any, len(decoded)+1)
	for k, v := range decoded {
		data[k] = v
	}
	data[tsl.PropertyDeviceId] = s.deviceId
	if err := product.GetTimeSeries().SaveProperties(product, data); err != nil {
		logs.Warnf("modbus SaveProperties device=%s err=%v", s.deviceId, err)
		s.debugLog("error", "collector SaveProperties: %v", err)
		return
	}
	s.lastCollect.Store(time.Now().UnixMilli())
}

func (s *ModbusSession) groupShouldReport(g Group, decoded map[string]any, types map[string]string) bool {
	report := g.Report
	if report == "" {
		report = ReportOnPeriod
	}
	switch report {
	case ReportOnPeriod:
		return true
	case ReportOnChange, ReportOnChangeOrPeriod:
		changed := false
		for k, v := range decoded {
			old, _ := s.lastValues.Load(k)
			if valueChanged(old, v, g.Deadband, isNumericType(types[k])) {
				changed = true
				break
			}
		}
		for k, v := range decoded {
			s.lastValues.Store(k, v)
		}
		if changed {
			s.heartbeatN.Store(g.Id, 0)
			return true
		}
		if report == ReportOnChangeOrPeriod && g.HeartbeatPeriods > 0 {
			n, _ := s.heartbeatN.LoadOrStore(g.Id, 0)
			cur := 0
			if iv, ok := n.(int); ok {
				cur = iv
			}
			cur++
			s.heartbeatN.Store(g.Id, cur)
			return cur >= g.HeartbeatPeriods
		}
		return false
	default:
		return true
	}
}

func (s *ModbusSession) bumpGroupFail(groupId string, cfg *CollectorConfig) {
	n, _ := s.failCount.LoadOrStore(groupId, 0)
	cur := 0
	if iv, ok := n.(int); ok {
		cur = iv
	}
	cur++
	s.failCount.Store(groupId, cur)
	if cfg != nil && cfg.OfflineAfterFailures > 0 && cur >= cfg.OfflineAfterFailures {
		logs.Warnf("modbus offlineAfterFailures device=%s group=%s n=%d", s.deviceId, groupId, cur)
		s.debugLog("error", "collector offlineAfterFailures group=%s n=%d", groupId, cur)
		_ = s.Disconnect()
	}
}

func (s *ModbusSession) resetGroupFail(groupId string) {
	s.failCount.Store(groupId, 0)
}

// CollectNow 人工触发。不走 skip-if-busy。
func (s *ModbusSession) CollectNow(groupId string) error {
	if s.stopped.Load() {
		return fmt.Errorf("session stopped")
	}
	product := core.GetProduct(s.productId)
	if product == nil {
		return fmt.Errorf("product not published")
	}
	cfg := GetCollector(s.productId)
	if !ShouldStartCollector(cfg) {
		return fmt.Errorf("collector is not active")
	}
	var groups []Group
	for _, g := range cfg.Groups {
		if groupId != "" && g.Id != groupId {
			continue
		}
		groups = append(groups, g)
	}
	if groupId != "" && len(groups) == 0 {
		return fmt.Errorf("group %s not found", groupId)
	}
	rt := s.runtime
	if rt == nil {
		rt = &CollectorRuntime{ctx: stdctx.Background()}
	}
	for _, g := range groups {
		rt.collectGroup(s, cfg, product, g, nil)
	}
	return nil
}
