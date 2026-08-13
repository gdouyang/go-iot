package timeseries

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/es"
	"go-iot/pkg/models"
	deviceDao "go-iot/pkg/models/device"

	logs "go-iot/pkg/logger"
)

const (
	retentionDeleteBatch = 20
	// 启动后首次清理延迟，避免与 bootstrap 抢 ES
	retentionFirstDelay = 30 * time.Second
)

// MonthIndexMeta 解析后的时序月索引元信息。
type MonthIndexMeta struct {
	Kind      string    // properties | devicelogs | event
	ProductID string    // event 时为 productId 前缀匹配用；完整解析时见 ParseTimeseriesMonthIndex
	EventID   string    // 仅 event；若 productId 含 '-' 可能无法可靠拆出，可为空
	YearMonth time.Time // 当月 1 日 00:00 本地时区
	IndexName string
}

// EffectiveRetentionMonths 解析有效保留月数。
// product 为 nil 或 <=0：跟随全局 global；product >0：使用产品配置。
// 返回值 <=0 表示不清理（通常来自全局关闭）。
func EffectiveRetentionMonths(global int, product *int) int {
	if product != nil && *product > 0 {
		return *product
	}
	return global
}

// CutoffMonth 计算保留窗口中最早应保留的自然月（当月 1 日）。
// retentionMonths=N>0 时保留 [cutoff, currentMonth] 共 N 个月；
// 删除 indexMonth < cutoff 的索引。N<=0 返回零值。
func CutoffMonth(now time.Time, retentionMonths int) time.Time {
	if retentionMonths <= 0 {
		return time.Time{}
	}
	cur := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	return cur.AddDate(0, -(retentionMonths - 1), 0)
}

// ShouldDelete 当 indexMonth 早于 cutoff 时删除；cutoff 零值不删。
func ShouldDelete(indexMonth, cutoff time.Time) bool {
	if cutoff.IsZero() || indexMonth.IsZero() {
		return false
	}
	// 规范化到月首再比较
	im := time.Date(indexMonth.Year(), indexMonth.Month(), 1, 0, 0, 0, 0, indexMonth.Location())
	cf := time.Date(cutoff.Year(), cutoff.Month(), 1, 0, 0, 0, 0, cutoff.Location())
	return im.Before(cf)
}

// ParseMonthSuffix 从索引名解析末尾 -YYYYMM。
func ParseMonthSuffix(name string) (time.Time, bool) {
	if len(name) < 8 {
		return time.Time{}, false
	}
	if name[len(name)-7] != '-' {
		return time.Time{}, false
	}
	ym := name[len(name)-6:]
	for _, c := range ym {
		if c < '0' || c > '9' {
			return time.Time{}, false
		}
	}
	t, err := time.ParseInLocation("200601", ym, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	// 校验月份合法（Parse 对 200613 会进位，额外检查）
	if t.Format("200601") != ym {
		return time.Time{}, false
	}
	return t, true
}

// ParseTimeseriesMonthIndex 解析时序月索引名；不合法返回 ok=false（永不删除）。
// event 类型：ProductID 为去掉前缀与月份后的整段 rest（productId-eventId），EventID 为空；
// 清理任务请用 MatchProductTimeseriesIndex 结合已知 productId 判断。
func ParseTimeseriesMonthIndex(name string) (MonthIndexMeta, bool) {
	month, ok := ParseMonthSuffix(name)
	if !ok {
		return MonthIndexMeta{}, false
	}
	base := name[:len(name)-7] // 去掉 -YYYYMM

	switch {
	case strings.HasPrefix(base, properties_const+"-"):
		pid := base[len(properties_const)+1:]
		if pid == "" {
			return MonthIndexMeta{}, false
		}
		return MonthIndexMeta{Kind: core.TIME_TYPE_PROP, ProductID: pid, YearMonth: month, IndexName: name}, true
	case strings.HasPrefix(base, devicelogs_const+"-"):
		pid := base[len(devicelogs_const)+1:]
		if pid == "" {
			return MonthIndexMeta{}, false
		}
		return MonthIndexMeta{Kind: core.TIME_TYPE_LOGS, ProductID: pid, YearMonth: month, IndexName: name}, true
	case strings.HasPrefix(base, event_const+"-"):
		rest := base[len(event_const)+1:]
		if rest == "" || !strings.Contains(rest, "-") {
			// event 至少 productId-eventId 两段；若无 '-' 仍允许 productId 为空事件？要求至少一段
			if rest == "" {
				return MonthIndexMeta{}, false
			}
		}
		return MonthIndexMeta{Kind: core.TIME_TYPE_EVENT, ProductID: rest, YearMonth: month, IndexName: name}, true
	default:
		return MonthIndexMeta{}, false
	}
}

// MatchProductTimeseriesIndex 在已知 productId 时判断索引是否属于该产品时序索引。
func MatchProductTimeseriesIndex(name, productID string) (MonthIndexMeta, bool) {
	if productID == "" {
		return MonthIndexMeta{}, false
	}
	productID = esIndexPart(productID)
	month, ok := ParseMonthSuffix(name)
	if !ok {
		return MonthIndexMeta{}, false
	}
	base := name[:len(name)-7]
	propsBase := properties_const + "-" + productID
	logsBase := devicelogs_const + "-" + productID
	eventPrefix := event_const + "-" + productID + "-"

	switch {
	case base == propsBase:
		return MonthIndexMeta{Kind: core.TIME_TYPE_PROP, ProductID: productID, YearMonth: month, IndexName: name}, true
	case base == logsBase:
		return MonthIndexMeta{Kind: core.TIME_TYPE_LOGS, ProductID: productID, YearMonth: month, IndexName: name}, true
	case strings.HasPrefix(base, eventPrefix):
		eventID := strings.TrimPrefix(base, eventPrefix)
		return MonthIndexMeta{Kind: core.TIME_TYPE_EVENT, ProductID: productID, EventID: eventID, YearMonth: month, IndexName: name}, true
	default:
		return MonthIndexMeta{}, false
	}
}

// CollectExpiredIndices 根据产品与 cutoff 收集应删除的索引名（不执行删除）。
func CollectExpiredIndices(productID string, cutoff time.Time, listIndices func(pattern string) ([]string, error)) ([]string, error) {
	if productID == "" || cutoff.IsZero() {
		return nil, nil
	}
	// 与 ES 索引命名一致：产品 ID 段必须小写
	productID = esIndexPart(productID)
	patterns := []string{
		fmt.Sprintf("%s-%s-*", properties_const, productID),
		fmt.Sprintf("%s-%s-*", devicelogs_const, productID),
		fmt.Sprintf("%s-%s-*", event_const, productID),
	}
	var expired []string
	seen := map[string]struct{}{}
	for _, p := range patterns {
		names, err := listIndices(p)
		if err != nil {
			return expired, fmt.Errorf("list indices %s: %w", p, err)
		}
		for _, name := range names {
			meta, ok := MatchProductTimeseriesIndex(name, productID)
			if !ok {
				logs.Warnf("es retention skip unparsable index: %s", name)
				continue
			}
			if !ShouldDelete(meta.YearMonth, cutoff) {
				continue
			}
			if _, dup := seen[name]; dup {
				continue
			}
			seen[name] = struct{}{}
			expired = append(expired, name)
		}
	}
	return expired, nil
}

// EsRetentionJob ES 时序过期月索引清理任务。
type EsRetentionJob struct {
	// GlobalMonths 全局保留月数
	GlobalMonths int
	// ListIndices 默认真 ES CatIndices
	ListIndices func(pattern string) ([]string, error)
	// DeleteIndices 默认真 ES DeleteIndex
	DeleteIndices func(names ...string) error
	// ListProducts 默认分页拉 storePolicy=es 的产品
	ListProducts func(ctx context.Context) ([]models.Product, error)
	// Now 可注入，测试用
	Now func() time.Time
}

func (j *EsRetentionJob) ensureDefaults() {
	if j.ListIndices == nil {
		j.ListIndices = es.CatIndices
	}
	if j.DeleteIndices == nil {
		j.DeleteIndices = es.DeleteIndex
	}
	if j.ListProducts == nil {
		j.ListProducts = listEsProducts
	}
	if j.Now == nil {
		j.Now = time.Now
	}
}

func listEsProducts(ctx context.Context) ([]models.Product, error) {
	_ = ctx
	var all []models.Product
	pageNum := 1
	const pageSize = 200
	for {
		page := &models.PageQuery{
			PageNum:  pageNum,
			PageSize: pageSize,
			Condition: []core.SearchTerm{
				{Key: "storePolicy", Value: core.TIME_SERISE_ES, Oper: core.EQ},
			},
		}
		res, err := deviceDao.PageProductAll(page)
		if err != nil {
			return all, err
		}
		if res == nil || len(res.List) == 0 {
			break
		}
		all = append(all, res.List...)
		if len(res.List) < pageSize {
			break
		}
		pageNum++
		if pageNum > 10000 {
			// 安全阀
			break
		}
	}
	return all, nil
}

// RunOnce 执行一轮清理。
func (j *EsRetentionJob) RunOnce(ctx context.Context) error {
	j.ensureDefaults()
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	products, err := j.ListProducts(ctx)
	if err != nil {
		return fmt.Errorf("list products: %w", err)
	}

	now := j.Now()
	var toDelete []string
	for _, p := range products {
		if p.StorePolicy != core.TIME_SERISE_ES {
			continue
		}
		n := EffectiveRetentionMonths(j.GlobalMonths, p.RetentionMonths)
		if n <= 0 {
			continue
		}
		cutoff := CutoffMonth(now, n)
		expired, err := CollectExpiredIndices(p.Id, cutoff, j.ListIndices)
		if err != nil {
			logs.Errorf("es retention product=%s collect: %v", p.Id, err)
			continue
		}
		toDelete = append(toDelete, expired...)
	}

	if len(toDelete) == 0 {
		logs.Debugf("es retention: nothing to delete (products=%d globalMonths=%d)", len(products), j.GlobalMonths)
		return nil
	}

	deleted := 0
	for i := 0; i < len(toDelete); i += retentionDeleteBatch {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		end := i + retentionDeleteBatch
		if end > len(toDelete) {
			end = len(toDelete)
		}
		batch := toDelete[i:end]
		if err := j.DeleteIndices(batch...); err != nil {
			logs.Errorf("es retention delete batch failed: %v indices=%v", err, batch)
			continue
		}
		deleted += len(batch)
		logs.Infof("es retention deleted %d indices: %v", len(batch), batch)
	}
	logs.Infof("es retention round done: deleted=%d candidates=%d products=%d", deleted, len(toDelete), len(products))
	return nil
}

// StartEsRetentionLoop 启动后台清理循环。checkHours<=0 不启动。
// 返回 cancel 函数。
func StartEsRetentionLoop(globalMonths int, checkHours int, stop <-chan struct{}) (cancel func()) {
	if checkHours <= 0 {
		logs.Infof("es retention job disabled (retention-check-hours=%d)", checkHours)
		return func() {}
	}
	ctx, cancelFn := context.WithCancel(context.Background())
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logs.Errorf("es retention loop panic: %v", r)
			}
		}()
		job := &EsRetentionJob{GlobalMonths: globalMonths}
		// 首次延迟
		timer := time.NewTimer(retentionFirstDelay)
		defer timer.Stop()
		select {
		case <-ctx.Done():
			return
		case <-stop:
			return
		case <-timer.C:
			if err := job.RunOnce(ctx); err != nil {
				logs.Errorf("es retention first run: %v", err)
			}
		}

		ticker := time.NewTicker(time.Duration(checkHours) * time.Hour)
		defer ticker.Stop()
		logs.Infof("es retention job started: interval=%dh globalMonths=%d", checkHours, globalMonths)
		for {
			select {
			case <-ctx.Done():
				return
			case <-stop:
				return
			case <-ticker.C:
				func() {
					defer func() {
						if r := recover(); r != nil {
							logs.Errorf("es retention run panic: %v", r)
						}
					}()
					// 每轮刷新全局配置（支持运行期改 DefaultEsConfig 时生效）
					job.GlobalMonths = es.DefaultEsConfig.RetentionMonths
					if err := job.RunOnce(ctx); err != nil {
						logs.Errorf("es retention run: %v", err)
					}
				}()
			}
		}
	}()
	return cancelFn
}
