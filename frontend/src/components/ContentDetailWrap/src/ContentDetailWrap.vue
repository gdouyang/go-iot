<script setup lang="ts">
import { ElCard } from 'element-plus'
import { propTypes } from '@/utils/propTypes'
import { useDesign } from '@/hooks/web/useDesign'

const { getPrefixCls } = useDesign()

const prefixCls = getPrefixCls('content-detail-wrap')

defineProps({
  title: propTypes.string.def(''),
  message: propTypes.string.def(''),
  headerBorder: propTypes.bool.def(true)
})
</script>

<template>
  <div :class="[`${prefixCls}-container`, 'relative']">
    <ElCard :class="[`${prefixCls}-body`, 'mb-15px']" shadow="never">
      <div class="pb-15px" :class="[headerBorder ? 'header-border' : '']">
        <slot name="header"></slot>
      </div>
      <slot></slot>
    </ElCard>
  </div>
</template>
<style lang="less" scoped>
.header-border {
  border-bottom: 1px solid var(--el-border-color);
}
</style>
