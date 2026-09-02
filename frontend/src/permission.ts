import router from './router'
import type { RouteRecordRaw } from 'vue-router'
import { useTitle } from '@/hooks/web/useTitle'
import { useNProgress } from '@/hooks/web/useNProgress'
import { usePermissionStoreWithOut } from '@/store/modules/permission'
import { usePageLoading } from '@/hooks/web/usePageLoading'
import { NO_REDIRECT_WHITE_LIST } from '@/constants'
import { useUserStoreWithOut } from '@/store/modules/user'

const { start, done } = useNProgress()

const { loadStart, loadDone } = usePageLoading()

import { getAnonLicenseStatus } from '@/views/sys/api'
import { ElNotification } from 'element-plus'

let licenseChecked = false
let isLicenseRedirectRequired = false

const hasLicensePermission = (userInfo: any) => {
  if (!userInfo) return false
  if (userInfo.username === 'admin') return true
  const perms = userInfo.permissions || []
  return (
    perms.includes('license-mgr') ||
    perms.includes('license-mgr:query') ||
    perms.includes('license-mgr:save')
  )
}

export const refreshLicenseStatus = async () => {
  try {
    const res = await getAnonLicenseStatus()
    isLicenseRedirectRequired = !!res?.requireRedirect
    licenseChecked = true
    return res
  } catch (_) {
    return null
  }
}

router.beforeEach(async (to, from, next) => {
  start()
  loadStart()
  const permissionStore = usePermissionStoreWithOut()
  const userStore = useUserStoreWithOut()
  if (userStore.getUserInfo) {
    if (to.path === '/login') {
      next({ path: '/' })
      return
    }

    // 检查系统 License 授权状态
    if (!licenseChecked || to.path === '/sys/license' || to.path === '/license-required') {
      await refreshLicenseStatus()
    }

    // 系统需要授权时的分流跳转：有权限去授权管理，无权限去提示页
    if (isLicenseRedirectRequired) {
      const hasPerm = hasLicensePermission(userStore.getUserInfo)
      if (hasPerm) {
        if (to.path !== '/sys/license') {
          ElNotification({
            title: '系统未授权',
            message:
              '当前系统尚未导入有效 License 授权文件或授权已过期，已为您自动跳转至授权页面。',
            type: 'warning',
            duration: 4500
          })
          next({ path: '/sys/license' })
          return
        }
      } else {
        if (to.path !== '/license-required') {
          next({ path: '/license-required' })
          return
        }
      }
    } else {
      // 已经正常授权时，禁止停留在无权限提示页
      if (to.path === '/license-required') {
        next({ path: '/' })
        return
      }
    }

    if (permissionStore.getIsAddRouters) {
      next()
      return
    }

    await permissionStore.generateRoutes('static')

    permissionStore.getAddRouters.forEach((route) => {
      router.addRoute(route as unknown as RouteRecordRaw) // 动态添加可访问路由表
    })
    const redirectPath = from.query.redirect || to.path
    const redirect = decodeURIComponent(redirectPath as string)
    const nextData = to.path === redirect ? { ...to, replace: true } : { path: redirect }
    permissionStore.setIsAddRouters(true)
    next(nextData)
  } else {
    if (NO_REDIRECT_WHITE_LIST.indexOf(to.path) !== -1) {
      next()
    } else {
      next(`/login?redirect=${to.path}`) // 否则全部重定向到登录页
    }
  }
})

router.afterEach((to) => {
  useTitle(to?.meta?.title as string)
  done() // 结束Progress
  loadDone()
})
