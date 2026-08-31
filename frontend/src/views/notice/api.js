import request from '@/axios'

export const configTableUrl = '/notifier/config/page'

export const historyTableUrl = '/notify/history/page'

export function remove(id) {
  return request.delete(`/notifier/config/${id}`)
}

export function get(id) {
  return request.get(`/notifier/config/${id}`)
}

export function start(id) {
  return request.post(`/notifier/config/${id}/start`)
}

export function stop(id) {
  return request.post(`/notifier/config/${id}/stop`)
}

export function addNotify(data) {
  return request.post(`/notifier/config`, data)
}

export function updateNotify(id, data) {
  return request.put(`/notifier/config/${id}`, data)
}

export function copyNotify(id) {
  return request.post(`/notifier/config/${id}/copy`)
}

export function configTypes() {
  return request.get(`/notifier/config/types`).then((res) => {
    return res.result
  })
}

export function testNotify(data) {
  return request.post(`/notifier/config/test`, data)
}

export function listAll(data) {
  return request.post(`/notifier/config/list`, data)
}
