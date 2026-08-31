import 'vue/jsx'

// 引入unocss
import '@/plugins/unocss'

// 导入全局的svg图标
import '@/plugins/svgIcon'

import { createApp } from 'vue'

import ElementPlus from 'element-plus'

import 'element-plus/dist/index.css'

// 初始化多语言
import { setupI18n } from '@/plugins/vueI18n'

// 引入状态管理
import { setupStore } from '@/store'

// 全局组件
import { setupGlobCom } from '@/components'

// 引入element-plus 图标库
import { setupElementPlusIcons } from '@/plugins/elementPlus'

// 引入全局样式
import '@/styles/index.less'

// 引入动画
import '@/plugins/animate.css'

// 路由
import { setupRouter } from './router'

// 权限
import { setupPermission } from './directives'

import App from './App.vue'

import './permission'

import Doc from '@/views/doc/Doc.vue'

// 创建实例
const setupAll = async () => {
  const app = createApp(App)

  await setupI18n(app)

  setupStore(app)

  setupGlobCom(app)

  app.use(ElementPlus)
  // 全局注册 @element-plus/icons-vue，模板中可直接使用 <el-icon><Edit /></el-icon>
  setupElementPlusIcons(app)

  app.component('Doc', Doc)

  setupRouter(app)

  setupPermission(app)

  app.mount('#app')
}

setupAll()
