import request from '@/axios'
import type { UserType } from './types'

export const loginApi = (data: UserType): Promise<IResponse<UserType>> => {
  return request.post('/login', data)
}

export const loginOutApi = (): Promise<IResponse> => {
  return request.post('/logout')
}
