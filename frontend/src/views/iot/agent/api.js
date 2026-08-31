import request from '@/axios'

export function getAgentStatus() {
  return request.get('/agent/status')
}

export function getAgentSettings() {
  return request.get('/agent/settings')
}

export function saveAgentSettings(data) {
  return request.put('/agent/settings', data)
}

export function createConversation(data) {
  return request.post('/agent/conversations', data)
}

export function pageConversations(data) {
  return request.post('/agent/conversations/page', data || { pageNum: 1, pageSize: 50 })
}

export function getConversation(id) {
  return request.get(`/agent/conversations/${id}`)
}

export function renameConversation(id, data) {
  return request.put(`/agent/conversations/${id}`, data)
}

export function deleteConversation(id) {
  return request.delete(`/agent/conversations/${id}`)
}

export function pageMessages(id) {
  return request.get(`/agent/conversations/${id}/messages`)
}

export function postMessage(id, data) {
  return request.post(`/agent/conversations/${id}/messages`, data)
}

export function continueRun(id) {
  return request.post(`/agent/conversations/${id}/continue`)
}

export function applyDrafts(id, data) {
  return request.post(`/agent/conversations/${id}/apply`, data)
}

export function rejectDrafts(id, data) {
  return request.post(`/agent/conversations/${id}/reject`, data)
}

export function cancelRun(id) {
  return request.post(`/agent/conversations/${id}/cancel`)
}
