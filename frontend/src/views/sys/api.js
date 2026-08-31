import request from '@/axios'

export const roleTableUrl = '/role/page'
export const userTableUrl = '/user/page'

export function getAllRole() {
  return request.post(`/role/page`, { pageNum: 1, pageSize: 10000, condition: [] })
}

export function getRole(id) {
  return request.get(`/role/${id}`)
}

export function editRole(data) {
  return request.put(`/role`, data)
}

export function addRole(data) {
  return request.post(`/role`, data)
}

export function removeRole(id) {
  return request.delete(`/role/${id}`)
}

export function getRoleRelMenus(id) {
  return request.get(`/role/ref-menus/${id}`)
}

export function getMenus() {
  return request.get(`/menu/list`)
}

export function getUser(id) {
  return request.get(`/user/${id}`)
}

export function editUser(data) {
  return request.put(`/user`, data)
}

export function addUser(data) {
  return request.post(`/user`, data)
}

export function enableUser(id) {
  return request.put(`/user/enable/${id}`)
}

export function disableUser(id) {
  return request.put(`/user/disable/${id}`)
}

export function removeUser(id) {
  return request.delete(`/user/${id}`)
}

export function saveSysConfig(data) {
  return request.post(`/system/config`, data)
}
export function getSysConfig() {
  return request.get(`/anon/system/config`).then((res) => res.result)
}

export function uploadFile(file) {
  return request.post(
    `/file/upload`,
    { file: file },
    {
      headers: { 'Content-Type': 'multipart/form-data' }
    }
  )
}
/** 获取用户信息 */
export function getUserInfo() {
  return request.get('/user-info').then((res) => res.result)
}

export function saveUserInfo(values) {
  return request.put('/user-info/save-basic', values)
}
export function updatePwd(values) {
  return request.put('user-info/update-pwd', values)
}
