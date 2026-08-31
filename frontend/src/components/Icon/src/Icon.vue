<script setup lang="ts">
import { computed, unref, type Component } from 'vue'
import { ElIcon } from 'element-plus'
import * as ElementPlusIconsVue from '@element-plus/icons-vue'
import { propTypes } from '@/utils/propTypes'
import { useDesign } from '@/hooks/web/useDesign'
import { Icon } from '@iconify/vue'

const { getPrefixCls } = useDesign()

const prefixCls = getPrefixCls('icon')

const props = defineProps({
  // icon name
  // - el:Plus / el:arrow-left：@element-plus/icons-vue 官方图标
  // - svg-icon:xxx：本地 svg
  // - 其他：Iconify（如 carbon: / ep: / ant-design:）
  icon: propTypes.string,
  // icon color
  color: propTypes.string,
  // icon size
  size: propTypes.number.def(16),
  hoverColor: propTypes.string
})

const toPascalCase = (name: string) =>
  name
    .split(/[-_]/)
    .filter(Boolean)
    .map((s) => s.charAt(0).toUpperCase() + s.slice(1))
    .join('')

/** 解析 @element-plus/icons-vue 组件，支持 el:Plus / el:arrow-left */
const elementPlusIcon = computed<Component | null>(() => {
  if (!props.icon) return null
  let name = ''
  if (props.icon.startsWith('el:')) {
    name = props.icon.slice(3)
  } else if (props.icon.startsWith('el-icon:')) {
    name = props.icon.slice(8)
  } else {
    return null
  }
  if (!name) return null
  // 已是 PascalCase 直接取；否则 kebab-case 转 PascalCase
  const key = /^[A-Z]/.test(name) ? name : toPascalCase(name)
  return ((ElementPlusIconsVue as Record<string, Component>)[key] as Component) || null
})

const isElementPlus = computed(() => !!elementPlusIcon.value)

const isLocal = computed(() => !!props.icon?.startsWith('svg-icon:'))

const symbolId = computed(() => {
  return unref(isLocal) ? `#icon-${props.icon.split('svg-icon:')[1]}` : props.icon
})

// 是否使用在线图标
const isUseOnline = computed(() => {
  return import.meta.env.VITE_USE_ONLINE_ICON === 'true'
})

const getIconifyStyle = computed(() => {
  const { color, size } = props
  return {
    fontSize: `${size}px`,
    color
  }
})
</script>

<template>
  <ElIcon :class="prefixCls" :size="size" :color="color">
    <component :is="elementPlusIcon" v-if="isElementPlus" />

    <svg v-else-if="isLocal" aria-hidden="true">
      <use :xlink:href="symbolId" />
    </svg>

    <template v-else>
      <Icon v-if="isUseOnline" :icon="icon" :style="getIconifyStyle" />
      <div v-else :class="`${icon} iconify`" :style="getIconifyStyle"></div>
    </template>
  </ElIcon>
</template>

<style lang="less" scoped>
@prefix-cls: ~'@{namespace}-icon';

.@{prefix-cls},
.iconify {
  :deep(svg) {
    &:hover {
      // stylelint-disable-next-line
      color: v-bind(hoverColor) !important;
    }
  }
}

.iconify {
  &:hover {
    // stylelint-disable-next-line
    color: v-bind(hoverColor) !important;
  }
}
</style>
