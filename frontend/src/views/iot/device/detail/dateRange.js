import dayjs from 'dayjs'

/**
 * 默认时间范围：最近 1 个月
 * 对齐后端 ES 时序查询 getQueryIndexs：未传 createTime 时默认扫描「上月 + 当月」索引
 * @see go-iot/pkg/timeseries/timeseries_es.go getQueryIndexs
 */
export function getDefaultCreateTimeRange() {
  const end = dayjs()
  const start = end.subtract(1, 'month')
  return [start.format('YYYY-MM-DD HH:mm:ss'), end.format('YYYY-MM-DD HH:mm:ss')]
}

/** 日期选择器默认时刻：起 00:00:00 / 止 23:59:59 */
export const defaultTime = [new Date(2000, 0, 1, 0, 0, 0), new Date(2000, 0, 1, 23, 59, 59)]

/** 日期快捷选项 */
export const dateShortcuts = [
  {
    text: '最近1天',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24)
      return [start, end]
    }
  },
  {
    text: '最近7天',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setTime(start.getTime() - 3600 * 1000 * 24 * 7)
      return [start, end]
    }
  },
  {
    text: '最近1个月',
    value: () => {
      const end = new Date()
      const start = new Date()
      start.setMonth(start.getMonth() - 1)
      return [start, end]
    }
  }
]
