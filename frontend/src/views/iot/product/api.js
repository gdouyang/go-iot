import request from '@/axios'

export const tableUrl = 'product/page'
/**
 * deploy product
 * @param {*} id
 */
export function deploy(id) {
  return request.post(`product/${id}/deploy`)
}
/**
 * undeploy product
 * @param {*} id
 */
export function undeploy(id) {
  return request.post(`product/${id}/undeploy`)
}
/**
 * delete product by id
 * @param {*} id
 */
export function remove(id) {
  return request.delete(`product/${id}`)
}

export function get(id) {
  return request.get(`/product/${id}`)
}

/** 可选时序存储策略（受服务端 tdengine.enabled 等影响） */
export function listStorePolicies() {
  return request.get(`/product/store-policies`)
}

export function getTsl(id) {
  return request.get(`/product/${id}/tsl`)
}

export function getMetaconfig(id) {
  return request.get(`/product/${id}`).then((resp) => {
    if (resp.success && resp.result) {
      return resp.result.metaconfig ? resp.result.metaconfig : []
    }
    return []
  })
}

export function modifyTsl(id, data) {
  return request.put(`/product/${id}/tsl`, data)
}

export function updateScript(id, data) {
  return request.put(`/product/${id}/script`, data)
}

export function addProduct(data) {
  return request.post(`/product`, data)
}

export function updateProduct(id, data) {
  return request.put(`/product/${id}`, data)
}

export function getNetwork(id) {
  return request.get(`/product/network/${id}`)
}

export function updateNetwork(saveData) {
  return request.put(`/product/network`, saveData)
}

export function runNetwork(productId, state) {
  return request.post(`product/network/${productId}/run?state=${state}`)
}

export function getExportUrl(productId) {
  return `api/product/${productId}/export`
}

export function uploadProduct(file) {
  return request.post(
    `product/import`,
    { file: file },
    {
      headers: { 'Content-Type': 'multipart/form-data' }
    }
  )
}

export function getEventBusUrl(productId, deviceId, type) {
  if (window.location.protocol.startsWith('https')) {
    return `wss://${window.location.host}/api/eventbus/${productId}/${deviceId}/${type}`
  }
  return `ws://${window.location.host}/api/eventbus/${productId}/${deviceId}/${type}`
}
