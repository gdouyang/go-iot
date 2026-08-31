import request from '@/axios'

export const pageUrl = 'device/page'
/**
 * 状态文本
 * @param status
 */
export function getStatusText(status) {
  return status === 'online' ? '在线' : status === 'offline' ? '离线' : '未激活'
}

// 分页查询
export function page(param) {
  return request.post(pageUrl, param)
}
// 激活
export function deploy(id) {
  return request.post(`device/${id}/deploy`)
}
// 停用
export function undeploy(id) {
  return request.post(`device/${id}/undeploy`)
}

export function batchDeploy(ids) {
  return request.post(`device/batch/deploy`, ids)
}

export function batchUndeploy(ids) {
  return request.post(`device/batch/undeploy`, ids)
}

export function remove(id) {
  return request.delete(`device/${id}`)
}

export function get(id) {
  return request.get(`/device/${id}`)
}

export function getDetail(id) {
  return request.get(`/device/${id}/detail`)
}
/**
 * 获取设备连接信息
 * @param {string} id
 * @returns
 */
export function getConnectionInfo(id) {
  return request.get(`/device/${id}/connection-info`)
}
/**
 * 设备诊断
 * @param {string} id
 * @returns
 */
export function connectionCheck(id) {
  return request.get(`/device/${id}/connection-check`)
}
/**
 * 连接设备
 * @param {string} deviceId
 * @returns
 */
export function connect(deviceId) {
  return request.post(`/device/${deviceId}/connect`)
}

export function disconnect(deviceId) {
  return request.post(`/device/${deviceId}/disconnect`)
}

export function addDevice(data) {
  return request.post('/device', data)
}

export function updateDevice(data) {
  return request.put('/device', data)
}

export function getProductList() {
  return request.get('/product/list')
}

export function cmdInvoke(deviceId, params) {
  return request.post(`/device/${deviceId}/invoke`, params)
}

export function updateLocation(params) {
  return request.post(`/device/location`, params)
}

export function queryProperty(deviceId, data) {
  return request.post(getDevicePropertysUrl(deviceId), data)
}

export function queryEvent(deviceId, eventId, data) {
  return request.post(getDeviceEventsUrl(deviceId, eventId), data)
}

export function queryLogs(deviceId, data) {
  return request.post(getDeviceLogsUrl(deviceId), data)
}
/**
 * 设备导入
 * @param {string} productId
 * @param {File} file
 * @returns
 */
export function importDevice(productId, file) {
  return request.post(
    `device/${productId}/import`,
    { file: file },
    {
      headers: { 'Content-Type': 'multipart/form-data' }
    }
  )
}

export function getDeviceLogsUrl(deviceId) {
  return `/device/${deviceId}/logs`
}

export function getDevicePropertysUrl(deviceId) {
  return `/device/${deviceId}/properties`
}

export function getDeviceEventsUrl(deviceId, eventId) {
  return `/device/${deviceId}/event/${eventId}`
}

export function getEventBusUrl(deviceId, topic) {
  if (window.location.protocol.startsWith('https')) {
    return `wss://${window.location.host}/api/eventbus/${deviceId}/${topic}`
  }
  return `ws://${window.location.host}/api/eventbus/${deviceId}/${topic}`
}

export function getTemplateDownloadUrl(productId) {
  return `api/device/${productId}/template`
}

export function getImportResultUrl(token) {
  return `api/device/import-result/${token}`
}

export const otaPageUrl = 'device-ota/page'

// OTA分页查询
export function otaPage(param) {
  return request.post(otaPageUrl, param)
}

// OTA Update
export function otaUpdate(file, deviceIds, chunkSize, timeout, otaFileId) {
  const formData = new FormData()
  if (file) {
    formData.append('file', file)
  }
  if (otaFileId) {
    formData.append('otaFileId', otaFileId)
  }
  formData.append('deviceIds', deviceIds)
  if (chunkSize) {
    formData.append('chunkSize', chunkSize)
  }
  if (timeout) {
    formData.append('timeout', timeout)
  }
  return request.post(`device-ota/add`, formData)
}

// OTA媒体列表
export function otaFileList(productId) {
  return otaFilePage({
    pageNum: 1,
    pageSize: 1000,
    condition: [{ key: 'productId', value: productId, oper: 'EQ' }]
  }).then((res) => {
    if (res && res.result && res.result.list) {
      return res.result.list
    }
    return []
  })
}

export const otaFilePageUrl = 'device-ota/file/page'

// OTA媒体分页
export function otaFilePage(params) {
  return request.post(otaFilePageUrl, params)
}

// OTA媒体新增
export function otaFileAdd(data) {
  return request.post('device-ota/file/add', data)
}

// OTA媒体更新
export function otaFileUpdate(data) {
  return request.put('device-ota/file/update', data)
}

// OTA媒体删除
export function otaFileDelete(ids) {
  return request.delete('device-ota/file/delete', { data: ids })
}

// 删除OTA记录
export function otaDelete(ids) {
  return request.delete(`device-ota/delete`, { data: ids })
}
