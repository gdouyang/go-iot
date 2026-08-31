<template>
  <div>
    <div class="editor" :class="{ 'full-screen': fullScreen }">
      <div class="toolbars">
        <div>
          <el-button link type="primary" @click="save" title="保存编解码脚本 Ctrl-S|Command-S"
            >保存</el-button
          >
          <el-button
            link
            type="primary"
            @click="formatCode"
            title="对代码排版 Ctrl-Shift-F|Command-Shift-F"
            >美化代码</el-button
          >
        </div>
        <div>
          <el-button link type="primary" @click="showDebug" title="查看console.log调试日志"
            >日志</el-button
          >
          <el-button link type="primary" @click="showDemo" title="查看查看样例代码与方法说明"
            >查看样例</el-button
          >
          <el-button link type="primary" @click="switchFullScreen">{{
            fullScreen ? '退出全屏' : '全屏'
          }}</el-button>
        </div>
      </div>
      <div v-if="network.type" class="ace-div">
        <AceEditor
          ref="AceEditor"
          v-model:value="script"
          lang="javascript"
          theme="tomorrow_night"
          :options="aceOptions"
          class="ace-div"
          @init="init"
        />
      </div>
    </div>

    <el-drawer
      v-if="openDrawer"
      :title="`脚本说明(${network.type})`"
      placement="right"
      :model-value="true"
      size="750"
      @close="openDrawer = false"
    >
      <Doc :type="network.type" />
    </el-drawer>
    <DebugLog ref="debugLog" :product-id="id" />
  </div>
</template>

<script lang="jsx">
import { getNetwork, updateScript } from '@/views/iot/product/api.js'
import ace from 'ace-builds'
import { VAceEditor as AceEditor } from 'vue3-ace-editor'

import 'ace-builds/src-noconflict/ext-language_tools'
import 'ace-builds/src-noconflict/ext-searchbox'
import 'ace-builds/src-noconflict/mode-javascript'
import 'ace-builds/src-noconflict/theme-tomorrow_night'
import 'ace-builds/src-noconflict/snippets/javascript'
import 'ace-builds/src-noconflict/ext-beautify'

// import _ from 'lodash-es'
import { addCompletions, getCompletions } from './codec/CodeCompletions.js'
import DebugLog from './DebugLog.vue'

export default {
  name: 'Codec1',
  components: {
    AceEditor,
    DebugLog
  },
  props: {
    id: {
      type: String,
      required: true
    },
    product: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      script: null,
      aceOptions: {
        enableBasicAutocompletion: true, // 启用基本自动完成功能
        enableLiveAutocompletion: true, // 启用实时自动完成功能 （比如：智能代码提示）
        enableSnippets: true, // 启用代码段
        showLineNumbers: true,
        tabSize: 2,
        wrapEnabled: true,
        showPrintMargin: true,
        readOnly: false,
        fontSize: 14
      },
      network: {},
      openDrawer: false,
      fullScreen: false
    }
  },
  computed: {
    isClientNet() {
      return (
        this.network.type === 'TCP_CLIENT' ||
        this.network.type === 'MQTT_CLIENT' ||
        this.network.type === 'MODBUS'
      )
    }
  },
  mounted() {
    this.open()
  },
  beforeUnmount() {
    if (this.editor) {
      this.editor.destroy()
    }
  },
  methods: {
    open() {
      this.script = this.product.script
      getNetwork(this.id).then((data) => {
        if (data.success) {
          this.network = data.result
          this.codeTip = getCompletions(this.network.type)
        }
      })
    },
    save() {
      if (!this.script) {
        this.$message.error('请填写物模型')
      }
      updateScript(this.id, { script: this.script }).then((data) => {
        if (data.success) {
          if (this.isClientNet) {
            this.$message.success('操作成功,重新连接后生效')
          } else {
            this.$message.success('操作成功')
          }
        }
      })
    },
    showDemo() {
      this.openDrawer = true
    },
    switchFullScreen() {
      this.fullScreen = !this.fullScreen
      setTimeout(() => {
        this.editor.resize(true)
      })
    },
    init(editor) {
      // 保存快捷键
      editor.commands.addCommands([
        {
          name: 'save',
          bindKey: { win: 'Ctrl-S', mac: 'Command-S' },
          exec: (arg) => {
            this.save()
          }
        },
        {
          name: 'beautify',
          bindKey: { win: 'Ctrl-Shift-F', mac: 'Command-Shift-F' },
          exec: (editor) => {
            this.formatCode()
          }
        }
      ])
      addCompletions(editor, this.codeTip)
      this.editor = editor
    },
    formatCode() {
      ace.define.modules['ace/ext/beautify'].beautify(this.editor.session)
    },
    change() {
      setTimeout(() => {
        this.$emit('editor-change', this.codeContent_)
      }, 10)
    },
    showDebug() {
      this.$refs.debugLog.open()
    }
  }
}
</script>
<style lang="less" scoped>
.editor {
  height: 100vh;
  width: 1000px;
  .ace-div {
    height: calc(100vh - 300px);
  }
  &.full-screen {
    position: fixed;
    top: 0;
    left: 0;
    z-index: 99;
    height: 100%;
    width: 100%;
    overflow: auto;
    .ace-div {
      height: calc(100% - 23px);
    }
  }
}
.toolbars {
  background: #eee;
  display: flex;
  justify-content: space-between;
  a {
    margin: 0 5px;
  }
}
</style>
