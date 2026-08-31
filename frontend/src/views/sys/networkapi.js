import request from '@/axios'
import { ElMessageBox } from 'element-plus'

export const tableUrl = '/server/page'

export function getAllNetwork() {
  return request.post(`/server/page`, { pageNum: 1, pageSize: 10000, condition: [] })
}

export function getNetwork(id) {
  return request.get(`/server/${id}`)
}

export function editNetwork(data) {
  return request.put(`/server`, data)
}

export function addNetwork(data) {
  return request.post(`/server`, data)
}

export function removeNetwork(id) {
  return request.delete(`/server/${id}`)
}

export function meters(id) {
  request.get(`server/meters/${id}`).then((data) => {
    const content = JSON.stringify(data.result, null, 2)
    ElMessageBox.alert(
      `<div><pre style="padding: 5px; background-color: #efefef;">${content}</pre></div>`,
      '信息',
      {
        dangerouslyUseHTMLString: true
      }
    )
  })
}
