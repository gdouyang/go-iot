import request from '@/axios'

export const tableUrl = '/rule/page'

export function remove(id) {
  return request.delete(`/rule/${id}`)
}

export function get(id) {
  return request.get(`/rule/${id}`)
}

export function start(id) {
  return request.post(`/rule/${id}/start`)
}

export function stop(id) {
  return request.post(`/rule/${id}/stop`)
}

export function addScene(data) {
  return request.post(`/rule`, data)
}

export function updateScene(id, data) {
  return request.put(`/rule/${id}`, data)
}

export function copy(id) {
  return request.post(`/rule/${id}/copy`)
}
