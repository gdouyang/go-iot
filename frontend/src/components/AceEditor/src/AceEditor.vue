<script setup lang="ts">
import { onBeforeUnmount, computed, PropType, unref, nextTick, ref, watch, shallowRef } from 'vue'
import { propTypes } from '@/utils/propTypes'
// import ace from 'ace-builds'
// import * as ace from 'brace'
// import './codec/beautify'
import { VAceEditor as AceEditor } from 'vue3-ace-editor'

import 'ace-builds/src-noconflict/ext-language_tools'
import 'ace-builds/src-noconflict/ext-searchbox'
import 'ace-builds/src-noconflict/mode-javascript'
import 'ace-builds/src-noconflict/theme-tomorrow_night'
import 'ace-builds/src-noconflict/snippets/javascript'

const props = defineProps({
  modelValue: propTypes.string.def('')
})

const emit = defineEmits(['init', 'update:modelValue'])

// 编辑器实例，必须用 shallowRef
const editorRef = shallowRef()

const valueHtml = ref('')

const aceOptions = {
  enableBasicAutocompletion: true, // 启用基本自动完成功能
  enableLiveAutocompletion: true, // 启用实时自动完成功能 （比如：智能代码提示）
  enableSnippets: true, // 启用代码段
  showLineNumbers: true,
  tabSize: 2,
  wrapEnabled: true,
  showPrintMargin: true,
  readOnly: false,
  fontSize: 14
}

watch(
  () => props.modelValue,
  (val: string) => {
    if (val === unref(valueHtml)) return
    valueHtml.value = val
  },
  {
    immediate: true
  }
)

// 监听
watch(
  () => valueHtml.value,
  (val: string) => {
    emit('update:modelValue', val)
  }
)

const handleCreated = (editor: any) => {
  editorRef.value = editor
  emit('init', editor)
}

// 组件销毁时，及时销毁编辑器
onBeforeUnmount(() => {
  const editor = unref(editorRef.value)

  // 销毁，并移除 editor
  editor?.destroy()
})

const getEditorRef = async (): Promise<any> => {
  await nextTick()
  return unref(editorRef.value) as any
}

defineExpose({
  getEditorRef
})
</script>

<template>
  <AceEditor
    ref="editorRef"
    v-bind="$attrs"
    v-model="valueHtml"
    lang="javascript"
    theme="tomorrow_night"
    :options="aceOptions"
    @init="handleCreated"
  />
</template>
