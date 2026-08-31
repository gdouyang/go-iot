<script setup lang="ts">
import { ElDialog, ElScrollbar } from 'element-plus'
import { propTypes } from '@/utils/propTypes'
import { computed, useAttrs, ref, unref, useSlots, watch, nextTick } from 'vue'
import { isNumber } from '@/utils/is'

const slots = useSlots()

const props = defineProps({
  title: propTypes.string.def('Dialog'),
  cancelText: propTypes.string.def('取消'),
  showOk: propTypes.bool.def(true),
  okBtnLoading: propTypes.bool.def(false),
  okText: propTypes.string.def('确定'),
  okType: propTypes.string.def('primary'),
  fullscreen: propTypes.bool.def(false),
  maxHeight: propTypes.oneOfType([String, Number]).def('500px')
})
const title_ = ref('')
const titleVal = computed(() => {
  if (title_.value) {
    return title_.value
  }
  return props.title
})
const emits = defineEmits(['close', 'confirm'])
const getBindValue = computed(() => {
  const delArr: string[] = ['fullscreen', 'title', 'maxHeight']
  const attrs = useAttrs()
  const obj = { ...attrs, ...props }
  for (const key in obj) {
    if (delArr.indexOf(key) !== -1) {
      delete obj[key]
    }
  }
  return obj
})

const isFullscreen = ref(false)
const modalStatus = ref(false)

const toggleFull = () => {
  isFullscreen.value = !unref(isFullscreen)
}

const dialogHeight = ref(isNumber(props.maxHeight) ? `${props.maxHeight}px` : props.maxHeight)

watch(
  () => isFullscreen.value,
  async (val: boolean) => {
    await nextTick()
    if (val) {
      const windowHeight = document.documentElement.offsetHeight
      dialogHeight.value = `${windowHeight - 55 - 60 - (slots.footer ? 63 : 0)}px`
    } else {
      dialogHeight.value = isNumber(props.maxHeight) ? `${props.maxHeight}px` : props.maxHeight
    }
  },
  {
    immediate: true
  }
)

const dialogStyle = computed(() => {
  return {
    height: unref(dialogHeight)
  }
})

const close = () => {
  modalStatus.value = false
  emits('close')
}

const ok = () => {
  emits('confirm')
}

const open = (config: any) => {
  modalStatus.value = true
  if (config) {
    title_.value = config.title
    if (config.onopen) {
      nextTick(() => {
        config.onopen()
      })
    }
  }
}
defineExpose({
  open: open,
  close: close
})
</script>

<template>
  <ElDialog
    v-model="modalStatus"
    v-bind="getBindValue"
    :fullscreen="isFullscreen"
    destroy-on-close
    lock-scroll
    draggable
    top="0"
    :close-on-click-modal="false"
    :show-close="false"
    @close="close"
  >
    <template #header="{ close }">
      <div class="flex justify-between items-center h-34px pl-15px pr-15px relative">
        <slot name="title">
          {{ titleVal }}
        </slot>
        <div
          class="h-34px flex justify-between items-center absolute top-[50%] right-15px translate-y-[-50%]"
        >
          <Icon
            v-if="fullscreen"
            class="cursor-pointer is-hover !h-34px mr-10px"
            :icon="isFullscreen ? 'radix-icons:exit-full-screen' : 'radix-icons:enter-full-screen'"
            color="var(--el-color-info)"
            hover-color="var(--el-color-primary)"
            @click="toggleFull"
          />
          <Icon
            class="cursor-pointer is-hover !h-34px"
            icon="ep:close"
            hover-color="var(--el-color-primary)"
            color="var(--el-color-info)"
            @click="close"
          />
        </div>
      </div>
    </template>

    <ElScrollbar :style="dialogStyle">
      <slot></slot>
    </ElScrollbar>

    <template #footer>
      <slot name="footer">
        <el-button @click="close">{{ cancelText }}</el-button>
        <el-button v-if="showOk" @click="ok" :type="okType" v-loading="okBtnLoading">{{
          okText
        }}</el-button>
      </slot>
    </template>
  </ElDialog>
</template>

<style lang="less">
.@{elNamespace}-overlay-dialog {
  display: flex;
  justify-content: center;
  align-items: center;
}

.@{elNamespace}-dialog {
  margin: 0 !important;
  &__header {
    margin-right: 0 !important;
    border-bottom: 1px solid var(--el-border-color);
    padding: 0;
    height: 34px;
  }
  &__body {
    padding: 15px !important;
  }
  &__footer {
    border-top: 1px solid var(--el-border-color);
  }
  &__headerbtn {
    top: 0;
  }
}
</style>
