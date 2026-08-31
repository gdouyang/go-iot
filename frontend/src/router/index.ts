import { createRouter, createWebHashHistory } from 'vue-router'
import type { RouteRecordRaw } from 'vue-router'
import type { App } from 'vue'
import { Layout, getParentLayout } from '@/utils/routerHelper'
import { useI18n } from '@/hooks/web/useI18n'

const { t } = useI18n()

export const constantRouterMap: AppRouteRecordRaw[] = [
  {
    path: '/',
    component: Layout,
    redirect: '/home',
    name: 'Root',
    meta: {
      hidden: true
    }
  },
  {
    path: '/redirect',
    component: Layout,
    name: 'Redirect',
    children: [
      {
        path: '/redirect/:path(.*)',
        name: 'RedirectPath',
        component: () => import('@/views/Redirect/Redirect.vue'),
        meta: {}
      }
    ],
    meta: {
      hidden: true,
      noTagsView: true
    }
  },
  {
    path: '/login',
    component: () => import('@/views/Login/Login.vue'),
    name: 'Login',
    meta: {
      hidden: true,
      title: t('router.login'),
      noTagsView: true
    }
  },
  {
    path: '/personal',
    component: Layout,
    redirect: '/personal/personal-center',
    name: 'Personal',
    meta: {
      title: t('router.personal'),
      hidden: true,
      canTo: true
    },
    children: [
      {
        path: 'personal-center',
        component: () => import('@/views/Personal/PersonalCenter/PersonalCenter.vue'),
        name: 'PersonalCenter',
        meta: {
          title: t('router.personalCenter'),
          hidden: true,
          canTo: true
        }
      }
    ]
  },
  {
    path: '/403',
    component: () => import('@/views/Error/403.vue'),
    name: 'NoPermission',
    meta: {
      hidden: true,
      title: '403',
      noTagsView: true
    }
  },
  {
    path: '/404',
    component: () => import('@/views/Error/404.vue'),
    name: 'NoFind',
    meta: {
      hidden: true,
      title: '404',
      noTagsView: true
    }
  }
]

export const asyncRouterMap: AppRouteRecordRaw[] = [
  {
    path: '/',
    component: Layout,
    name: 'Home',
    meta: {},
    children: [
      {
        path: '/home',
        name: 'Home1',
        component: () => import('@/views/Home/Home.vue'),
        meta: {
          title: t('首页'),
          icon: 'el:HomeFilled'
        }
      }
    ]
  },
  {
    path: '/product',
    component: Layout,
    name: 'Product',
    meta: {},
    children: [
      {
        path: 'product-list',
        name: 'ProductList',
        component: () => import('@/views/iot/product/ProductList.vue'),
        meta: { title: '产品管理', icon: 'el:Box', permission: ['product-mgr'] }
      }
    ]
  },
  {
    path: '/agent',
    component: Layout,
    name: 'Agent',
    meta: {},
    children: [
      {
        path: 'chat',
        name: 'AgentChat',
        component: () => import('@/views/iot/agent/AgentChat.vue'),
        meta: { title: 'AI 助手', icon: 'el:ChatDotRound', permission: ['agent-mgr'] }
      }
    ]
  },
  {
    path: '/device',
    component: Layout,
    name: 'Device',
    meta: {},
    children: [
      {
        path: 'device-list',
        name: 'DeviceList',
        component: () => import('@/views/iot/device/DeviceList.vue'),
        meta: { title: '设备管理', icon: 'el:Connection', permission: ['device-mgr'] }
      }
    ]
  },
  {
    path: '/device-ota',
    component: Layout,
    name: 'DeviceOta',
    meta: {},
    children: [
      {
        path: 'ota-list',
        name: 'OtaList',
        component: () => import('@/views/iot/device/OtaList.vue'),
        meta: { title: 'OTA升级', icon: 'el:Upload', permission: ['ota-mgr'] }
      }
    ]
  },
  {
    path: '/rule',
    component: Layout,
    name: 'Rule',
    meta: {},
    children: [
      {
        path: 'rule-list',
        name: 'RuleList',
        component: () => import('@/views/iot/rule/RuleList.vue'),
        meta: { title: '规则引擎', icon: 'el:SetUp', permission: ['rule-mgr'] }
      }
    ]
  },
  {
    path: '/alarm',
    component: Layout,
    name: 'Alarm',
    meta: {},
    children: [
      {
        path: 'alarm-list',
        name: 'AlarmList',
        component: () => import('@/views/iot/alarm/AlarmList.vue'),
        meta: { title: '设备告警', icon: 'el:Warning', permission: ['alarm-mgr'] }
      }
    ]
  },
  {
    path: '/notice',
    component: Layout,
    name: 'Notice',
    meta: {},
    children: [
      {
        path: 'config-list',
        name: 'ConfigList',
        component: () => import('@/views/notice/config/ConfigList.vue'),
        meta: { title: '通知配置', icon: 'el:Bell', permission: ['notify-config'] }
      }
    ]
  },
  {
    path: '/sys',
    name: 'sysPage',
    component: Layout,
    meta: {
      title: '系统管理',
      icon: 'el:Setting',
      alwaysShow: true
    },
    children: [
      {
        path: 'role-list',
        name: 'RoleList',
        component: () => import('@/views/sys/RoleList.vue'),
        meta: { title: '角色列表', permission: ['role-mgr'] }
      },
      {
        path: 'user-list',
        name: 'UserList',
        component: () => import('@/views/sys/UserList.vue'),
        meta: { title: '用户列表', keepAlive: true, permission: ['user-mgr'] }
      },
      {
        path: 'network-list',
        name: 'NetWorkList',
        component: () => import('@/views/sys/NetworkList.vue'),
        meta: { title: '网络管理', keepAlive: true, permission: ['network-config'] }
      },
      {
        path: 'system-config',
        name: 'SystemConfig',
        component: () => import('@/views/sys/config/SysConfig.vue'),
        meta: { title: '系统配置', keepAlive: true, permission: ['sys-config'] }
      }
    ]
  }
]

const router = createRouter({
  history: createWebHashHistory(),
  strict: true,
  routes: constantRouterMap as RouteRecordRaw[],
  scrollBehavior: () => ({ left: 0, top: 0 })
})

export const resetRouter = (): void => {
  const resetWhiteNameList = [
    'Redirect',
    'Login',
    'NoFind',
    'NoPermission',
    'Root',
    'Personal',
    'PersonalCenter'
  ]
  router.getRoutes().forEach((route) => {
    const { name } = route
    if (name && !resetWhiteNameList.includes(name as string)) {
      router.hasRoute(name) && router.removeRoute(name)
    }
  })
}

export const setupRouter = (app: App<Element>) => {
  app.use(router)
}

export default router
