import { PATH_URL } from '@/axios/service'
import { useUserStoreWithOut } from '@/store/modules/user'

export function joinApi(path) {
  const base = String(PATH_URL || '').replace(/\/$/, '')
  const p = String(path || '').replace(/^\//, '')
  return `${base}/${p}`
}

export async function postAgentStream(path, body, { onEvent, signal } = {}) {
  const userStore = useUserStoreWithOut()
  const tokenKey = userStore.getTokenKey || 'Authorization'
  const headers = {
    'Content-Type': 'application/json',
    Accept: 'text/event-stream',
    [tokenKey]: userStore.getToken || ''
  }
  const resp = await fetch(joinApi(path), {
    method: 'POST',
    headers,
    body: JSON.stringify(body || {}),
    signal
  })
  const ctype = resp.headers.get('content-type') || ''
  if (!ctype.includes('text/event-stream')) {
    const json = await resp.json().catch(() => ({}))
    if (!resp.ok || json.success === false) {
      const err = new Error(json.message || json.msg || `HTTP ${resp.status}`)
      err.response = { data: json, status: resp.status }
      throw err
    }
    return json
  }
  if (!resp.ok) {
    const json = await resp.json().catch(() => ({}))
    const err = new Error(json.message || json.msg || `HTTP ${resp.status}`)
    err.response = { data: json, status: resp.status }
    throw err
  }
  const reader = resp.body.getReader()
  const decoder = new TextDecoder()
  let buf = ''
  while (true) {
    const { done, value } = await reader.read()
    if (done) break
    buf += decoder.decode(value, { stream: true })
    const parts = buf.split('\n\n')
    buf = parts.pop() || ''
    for (const block of parts) {
      let event = 'message'
      let data = ''
      for (const line of block.split('\n')) {
        if (line.startsWith('event:')) event = line.slice(6).trim()
        if (line.startsWith('data:')) data += line.slice(5).trim()
      }
      if (data && onEvent) {
        try {
          onEvent(event, JSON.parse(data))
        } catch {
          onEvent(event, data)
        }
      }
    }
  }
  return { streamed: true }
}
