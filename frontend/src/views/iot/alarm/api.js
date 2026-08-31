import request from '@/axios'

export const pageUrl = '/alarm/log/page'

export function solveAlarmLog(id, data) {
  return request.put(`/alarm/log/${id}/solve`, data)
}
