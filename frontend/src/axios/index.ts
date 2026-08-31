import service from './service'
import { CONTENT_TYPE } from '@/constants'
import { useUserStoreWithOut } from '@/store/modules/user'

const request = (option: AxiosConfig) => {
  const { url, method, params, data, headers, responseType } = option
  const userStore = useUserStoreWithOut()
  if (headers) {
    if (!headers['Content-Type']) {
      headers['Content-Type'] = CONTENT_TYPE
    }
  }
  return service.request({
    url: url,
    method,
    params,
    data,
    responseType: responseType,
    headers: {
      [userStore.getTokenKey ?? 'Authorization']: userStore.getToken ?? '',
      ...headers
    }
  })
}

export default {
  get: <T = any>(url: string, option?: AxiosConfig) => {
    return request({ method: 'get', url: url, ...option }) as Promise<IResponse<T>>
  },
  post: <T = any>(url: string, data?: any, option?: AxiosConfig) => {
    return request({ method: 'post', url: url, data: data, ...option }) as Promise<IResponse<T>>
  },
  delete: <T = any>(url: string, option?: AxiosConfig) => {
    return request({ method: 'delete', url: url, ...option }) as Promise<IResponse<T>>
  },
  put: <T = any>(url: string, data?: any, option?: AxiosConfig) => {
    return request({ method: 'put', url: url, data: data, ...option }) as Promise<IResponse<T>>
  },
  cancelRequest: (url: string | string[]) => {
    return service.cancelRequest(url)
  },
  cancelAllRequest: () => {
    return service.cancelAllRequest()
  }
}
