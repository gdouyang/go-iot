<template>
  <Dialog
    ref="addModal"
    @confirm="addConfirm"
    @close="addClose"
    :width="550"
    maxHeight="auto"
    :showOk="showOk"
    cancelText="关闭"
  >
    <el-alert
      style="margin-bottom: 10px"
      title="高危操作警告：导入物模型将全量覆盖现存的属性、功能与事件！"
      description="覆盖后若删减或变更了已有标识，现网在线设备将无法正常解析入库，底层时序数据库映射及已配置的规则引擎可能立即异常，请务必提前备份。"
      type="error"
      show-icon
      v-if="showOk"
      :closable="false"
    />
    <AceEditor
      ref="AceEditor"
      v-model:value="tsl"
      lang="json"
      theme="chrome"
      style="width: 480px; height: 450px"
      :options="aceOptions"
      @init="init"
    />
  </Dialog>
</template>

<script lang="jsx">
import { VAceEditor as AceEditor } from 'vue3-ace-editor'
import 'ace-builds/src-noconflict/mode-json'
import 'ace-builds/src-noconflict/theme-chrome'
import _ from 'lodash-es'

export default {
  name: 'TslImportDialog',
  components: {
    AceEditor
  },
  data() {
    return {
      tsl: null,
      aceOptions: {
        enableBasicAutocompletion: true, // 启用基本自动完成功能
        enableLiveAutocompletion: true, // 启用实时自动完成功能 （比如：智能代码提示）
        enableSnippets: true, // 启用代码段
        showLineNumbers: true,
        tabSize: 2,
        wrapEnabled: true,
        showPrintMargin: true,
        readOnly: true
      },
      showOk: false
    }
  },
  created() {},
  methods: {
    open(tsl, isImport) {
      if (isImport) {
        this.showOk = true
        this.$refs.addModal.open({ title: '导入物模型' })
        this.tsl = ''
      } else {
        this.showOk = false
        this.$refs.addModal.open({ title: '查看物模型' })
        this.tsl = tsl ? JSON.stringify(tsl, null, 2) : ''
      }
    },
    init(editor) {
      editor.setOptions({
        readOnly: false
      })
    },
    validateTslJson(tsl) {
      const checkEnumList = (list, typeName) => {
        if (!Array.isArray(list)) return null
        for (const item of list) {
          if (!item) continue
          if (item.type === 'enum') {
            const elements = (item.elements || []).filter(
              (e) => e && e.value !== '' && e.value !== null && e.value !== undefined
            )
            if (!elements.length) {
              return `${typeName} [${item.name || item.id || '未命名'}] 为枚举类型(enum)，但枚举项(elements)为空，必须至少配置一个有效的枚举项`
            }
          }
        }
        return null
      }

      const checkObjectList = (list, typeName) => {
        if (!Array.isArray(list)) return null
        for (const item of list) {
          if (!item) continue
          if (item.type === 'object') {
            if (!Array.isArray(item.properties) || !item.properties.length) {
              return `${typeName} [${item.name || item.id || '未命名'}] 为结构体类型(object)，但子属性(properties)为空，必须至少包含一个参数`
            }
          }
        }
        return null
      }

      const propErr =
        checkEnumList(tsl.properties, '属性') || checkObjectList(tsl.properties, '属性')
      if (propErr) return propErr

      const eventErr = checkEnumList(tsl.events, '事件') || checkObjectList(tsl.events, '事件')
      if (eventErr) return eventErr

      if (Array.isArray(tsl.functions)) {
        for (const fn of tsl.functions) {
          if (!fn) continue
          const inputErr =
            checkEnumList(fn.inputs, `功能[${fn.name || fn.id}]输入参数`) ||
            checkObjectList(fn.inputs, `功能[${fn.name || fn.id}]输入参数`)
          if (inputErr) return inputErr
          if (fn.output && fn.output.type === 'enum') {
            const elements = (fn.output.elements || []).filter(
              (e) => e && e.value !== '' && e.value !== null && e.value !== undefined
            )
            if (!elements.length) {
              return `功能 [${fn.name || fn.id}] 的输出参数为枚举类型(enum)，但枚举项(elements)为空`
            }
          }
          if (fn.output && fn.output.type === 'object') {
            if (!Array.isArray(fn.output.properties) || !fn.output.properties.length) {
              return `功能 [${fn.name || fn.id}] 的输出参数为结构体类型(object)，但子属性(properties)为空`
            }
          }
        }
      }
      return null
    },
    addConfirm() {
      if (!this.tsl) {
        this.$message.error('请填写物模型')
        return
      }
      try {
        const tsl = JSON.parse(this.tsl)
        const text = JSON.stringify(tsl)
        if (!_.isObject(tsl) || !_.startsWith(text, '{') || !_.endsWith(text, '}')) {
          this.$message.error('物模型格式错误，请以“{”开始，以“}”结束')
          return
        }
        const enumError = this.validateTslJson(tsl)
        if (enumError) {
          this.$message.error(enumError)
          return
        }
        this.$emit('import', tsl)
        this.$refs.addModal.close()
      } catch (error) {
        this.$message.error('物模型格式错误，请保证为json字符串')
      }
    },
    addClose() {}
  }
}
</script>

<style lang="less"></style>
