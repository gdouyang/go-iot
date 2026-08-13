package timeseries

import (
	"context"
	"sort"
	"testing"
	"time"

	"go-iot/pkg/core"
	"go-iot/pkg/models"

	"github.com/stretchr/testify/require"
)

func TestEffectiveRetentionMonths(t *testing.T) {
	// nil / 0 / 负数：跟随全局
	require.Equal(t, 3, EffectiveRetentionMonths(3, nil))
	zero := 0
	require.Equal(t, 3, EffectiveRetentionMonths(3, &zero))
	neg := -1
	require.Equal(t, 3, EffectiveRetentionMonths(3, &neg))
	// 全局关闭时，产品未覆盖仍不清理
	require.Equal(t, 0, EffectiveRetentionMonths(0, nil))
	require.Equal(t, 0, EffectiveRetentionMonths(0, &zero))
	// 产品 >0 覆盖全局
	twelve := 12
	require.Equal(t, 12, EffectiveRetentionMonths(3, &twelve))
	require.Equal(t, 12, EffectiveRetentionMonths(0, &twelve))
}

func TestCutoffMonth(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.Local)
	require.True(t, CutoffMonth(now, 0).IsZero())
	require.True(t, CutoffMonth(now, -1).IsZero())

	c1 := CutoffMonth(now, 1)
	require.Equal(t, "202608", c1.Format("200601"))

	c3 := CutoffMonth(now, 3)
	require.Equal(t, "202606", c3.Format("200601"))
}

func TestShouldDelete(t *testing.T) {
	cutoff := time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local)
	require.False(t, ShouldDelete(time.Time{}, cutoff))
	require.False(t, ShouldDelete(cutoff, time.Time{}))
	require.False(t, ShouldDelete(time.Date(2026, 6, 1, 0, 0, 0, 0, time.Local), cutoff))
	require.False(t, ShouldDelete(time.Date(2026, 7, 1, 0, 0, 0, 0, time.Local), cutoff))
	require.True(t, ShouldDelete(time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local), cutoff))
}

func TestParseMonthSuffix(t *testing.T) {
	tm, ok := ParseMonthSuffix("goiot-properties-p1-202608")
	require.True(t, ok)
	require.Equal(t, "202608", tm.Format("200601"))

	_, ok = ParseMonthSuffix("goiot-properties-p1")
	require.False(t, ok)
	_, ok = ParseMonthSuffix("goiot-properties-p1-202613")
	require.False(t, ok)
	_, ok = ParseMonthSuffix("short")
	require.False(t, ok)
}

func TestParseTimeseriesMonthIndex(t *testing.T) {
	m, ok := ParseTimeseriesMonthIndex("goiot-properties-prod1-202601")
	require.True(t, ok)
	require.Equal(t, core.TIME_TYPE_PROP, m.Kind)
	require.Equal(t, "prod1", m.ProductID)

	m, ok = ParseTimeseriesMonthIndex("goiot-devicelogs-prod1-202601")
	require.True(t, ok)
	require.Equal(t, core.TIME_TYPE_LOGS, m.Kind)

	m, ok = ParseTimeseriesMonthIndex("goiot-event-prod1-fire_alarm-202601")
	require.True(t, ok)
	require.Equal(t, core.TIME_TYPE_EVENT, m.Kind)
	require.Equal(t, "prod1-fire_alarm", m.ProductID)

	_, ok = ParseTimeseriesMonthIndex("goiot-device-xxx-202601")
	require.False(t, ok)
}

func TestMatchProductTimeseriesIndex(t *testing.T) {
	m, ok := MatchProductTimeseriesIndex("goiot-event-ab-cd-fire-202603", "ab-cd")
	require.True(t, ok)
	require.Equal(t, "ab-cd", m.ProductID)
	require.Equal(t, "fire", m.EventID)
	require.Equal(t, "202603", m.YearMonth.Format("200601"))

	_, ok = MatchProductTimeseriesIndex("goiot-properties-other-202603", "ab-cd")
	require.False(t, ok)
}

func TestCollectExpiredIndices(t *testing.T) {
	now := time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local)
	cutoff := CutoffMonth(now, 3) // 202606
	pid := "p1"
	list := func(pattern string) ([]string, error) {
		switch {
		case pattern == "goiot-properties-p1-*":
			return []string{
				"goiot-properties-p1-202605",
				"goiot-properties-p1-202606",
				"goiot-properties-p1-202608",
			}, nil
		case pattern == "goiot-devicelogs-p1-*":
			return []string{"goiot-devicelogs-p1-202604"}, nil
		case pattern == "goiot-event-p1-*":
			return []string{"goiot-event-p1-e1-202605", "goiot-event-p1-e1-202607"}, nil
		default:
			return nil, nil
		}
	}
	got, err := CollectExpiredIndices(pid, cutoff, list)
	require.NoError(t, err)
	sort.Strings(got)
	require.Equal(t, []string{
		"goiot-devicelogs-p1-202604",
		"goiot-event-p1-e1-202605",
		"goiot-properties-p1-202605",
	}, got)
}

func TestEsRetentionJobRunOnce(t *testing.T) {
	now := time.Date(2026, 8, 10, 0, 0, 0, 0, time.Local)
	var deleted []string
	rm12 := 12
	job := &EsRetentionJob{
		GlobalMonths: 3,
		Now:          func() time.Time { return now },
		ListProducts: func(ctx context.Context) ([]models.Product, error) {
			return []models.Product{
				{Id: "a", StorePolicy: core.TIME_SERISE_ES},                           // nil → 跟随全局 3
				{Id: "b", StorePolicy: core.TIME_SERISE_ES, RetentionMonths: intPtr(0)}, // 0 → 跟随全局 3
				{Id: "c", StorePolicy: core.TIME_SERISE_MOCK},
				{Id: "d", StorePolicy: core.TIME_SERISE_ES, RetentionMonths: &rm12}, // 12 覆盖
			}, nil
		},
		ListIndices: func(pattern string) ([]string, error) {
			// a/b 用全局 3 月：202605 过期
			// d 用 12 月：202605 不过期
			if pattern == "goiot-properties-a-*" {
				return []string{"goiot-properties-a-202605", "goiot-properties-a-202608"}, nil
			}
			if pattern == "goiot-properties-b-*" {
				return []string{"goiot-properties-b-202605"}, nil
			}
			if pattern == "goiot-properties-d-*" {
				return []string{"goiot-properties-d-202605"}, nil
			}
			return []string{}, nil
		},
		DeleteIndices: func(names ...string) error {
			deleted = append(deleted, names...)
			return nil
		},
	}
	require.NoError(t, job.RunOnce(context.Background()))
	sort.Strings(deleted)
	require.Equal(t, []string{"goiot-properties-a-202605", "goiot-properties-b-202605"}, deleted)
}

func intPtr(v int) *int { return &v }
