<template>
  <div>
    <Dialog
      ref="addModal"
      title="TCP配置"
      width="500"
      maxHeight="auto"
      @confirm="addConfirm"
      @close="addClose"
    >
      <el-form ref="addFormRef" :model="addObj" style="width: 90%" label-width="auto">
        <el-form-item label="开启SSL" prop="configuration.useTLS">
          <el-radio-group v-model="addObj.configuration.useTLS">
            <el-radio :value="true">是</el-radio>
            <el-radio :value="false">否</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item
          v-if="addObj.configuration.useTLS"
          label="crt文件"
          prop="certBase64"
          :rules="[{ required: true, message: 'crt文件不能为空', trigger: 'blur' }]"
        >
          <CertificateUpload v-model="addObj.certBase64" />
          <el-input type="textarea" v-model="addObj.certBase64" show-word-limit />
        </el-form-item>
        <el-form-item
          v-if="addObj.configuration.useTLS"
          label="key文件"
          prop="keyBase64"
          :rules="[{ required: true, message: 'key文件不能为空', trigger: 'blur' }]"
        >
          <CertificateUpload v-model="addObj.keyBase64" />
          <el-input type="textarea" v-model="addObj.keyBase64" show-word-limit />
        </el-form-item>
        <el-form-item
          label="解析方式"
          prop="configuration.delimeter.type"
          :rules="[{ required: true, message: '请选择' }]"
        >
          <el-select v-model="addObj.configuration.delimeter.type" @change="parserTypeChange">
            <el-option value="NONE" label="不处理" />
            <el-option value="Delimited" label="分隔符" />
            <el-option value="FixLength" label="固定长度" />
            <el-option value="SplitFunc" label="自定义脚本" />
          </el-select>
        </el-form-item>
        <el-form-item
          label="分隔符"
          prop="configuration.delimeter.delimited"
          v-if="addObj.configuration.delimeter.type === 'Delimited'"
        >
          <el-input v-model="addObj.configuration.delimeter.delimited" :maxlength="64" />
        </el-form-item>
        <!-- 固定长度 -->
        <el-form-item
          label="长度值"
          prop="configuration.delimeter.length"
          v-if="addObj.configuration.delimeter.type === 'FixLength'"
        >
          <el-input-number v-model="addObj.configuration.delimeter.length" :min="0" :max="60000" />
        </el-form-item>
        <!-- 自定义脚本 -->
        <template v-if="addObj.configuration.delimeter.type === 'SplitFunc'">
          <el-form-item label="解析脚本" prop="configuration.delimeter.size">
            <el-button link type="primary" @click="showDemo">查看样例</el-button>
            <div class="ace-div">
              <VAceEditor
                ref="AceEditor"
                v-model:value="addObj.configuration.delimeter.splitFunc"
                lang="javascript"
                theme="chrome"
                :options="aceOptions"
                class="ace-div"
                @init="init"
              />
            </div>
          </el-form-item>
        </template>
      </el-form>
    </Dialog>
    <el-drawer
      v-model="openDrawer"
      title="说明"
      placement="right"
      size="50%"
      @close="openDrawer = false"
    >
      <Doc :type="'SplitFunc'" />
    </el-drawer>
  </div>
</template>

<script lang="jsx">
import { VAceEditor } from 'vue3-ace-editor'
import 'ace-builds/src-noconflict/ext-language_tools'
import 'ace-builds/src-noconflict/ext-searchbox'
import 'ace-builds/src-noconflict/mode-javascript'
import 'ace-builds/src-noconflict/theme-chrome'
import 'ace-builds/src-noconflict/snippets/javascript'

import _ from 'lodash-es'
import { newTcpAddObj } from './entity.js'
import Base from './Base.vue'

export default {
  name: 'TcpConfigAdd',
  components: {
    VAceEditor
  },
  mixins: [Base],
  props: {},
  data() {
    return {
      addObj: newTcpAddObj(),
      isEdit: false,
      aceOptions: {
        enableBasicAutocompletion: true, // 启用基本自动完成功能
        enableLiveAutocompletion: true, // 启用实时自动完成功能 （比如：智能代码提示）
        enableSnippets: true, // 启用代码段
        showLineNumbers: true,
        tabSize: 2,
        wrapEnabled: true,
        showPrintMargin: true,
        readOnly: false
      },
      openDrawer: false
    }
  },
  computed: {},
  created() {},
  methods: {
    open(productId) {
      if (!productId) {
        this.$message.error('请指定产品ID')
        return
      }
      this.productId = productId
      this.getNetwork(productId, newTcpAddObj()).then((data) => {
        this.isEdit = false
        if (data.id) {
          this.isEdit = true
        }
        this.addObj = data
        this.$refs.addModal.open()
      })
    },
    addClose() {
      this.addObj = newTcpAddObj()
      this.$refs.addFormRef.clearValidate()
    },
    addConfirm() {
      this.$refs.addFormRef.validate((valid) => {
        if (valid) {
          const saveData = _.cloneDeep(this.addObj)
          this.updateNetwork(saveData).then((resp) => {
            if (resp.success) {
              this.$message.success('操作成功')
              this.$refs.addModal.close()
              this.$emit('success')
            } else {
              this.$message.error(resp.message)
            }
          })
        }
      })
    },
    parserTypeChange(value) {
      const c = this.addObj.configuration.delimeter
      if (value === 'DIRECT') {
        c.delimited = undefined
        c.length = undefined
        c.splitFunc = undefined
      } else if (value === 'Delimited') {
        c.length = undefined
        c.splitFunc = undefined
      } else if (value === 'FixLength') {
        c.delimited = undefined
        c.splitFunc = undefined
      } else if (value === 'SplitFunc') {
        c.delimited = undefined
        c.length = undefined
        c.splitFunc = `function splitFunc(parser) {

}`
      }
    },
    showDemo() {
      this.openDrawer = true
    },
    init(editor) {
      // 加入自定义语法提示
      const that = this
      editor.completers = [
        {
          getCompletions: function (editor, session, pos, prefix, callback) {
            that.setCompletions(editor, session, pos, prefix, callback)
          }
        }
      ]
      this.editor = editor
    },
    setCompletions(editor, session, pos, prefix, callback) {
      if (prefix.length === 0) {
        return callback(null, [])
      } else {
        return callback(null, [
          { caption: 'parser.AddHandler()', value: 'parser.AddHandler(function(data){\n\n})' },
          { caption: 'parser.AppendResult()', value: 'parser.AppendResult(data)' },
          { caption: 'parser.Complete()', value: 'parser.Complete()' },
          { caption: 'parser.Fixed()', value: 'parser.Fixed(21)' },
          { caption: 'parser.Delimited()', value: 'parser.Delimited("\\n")' }
        ])
      }
    }
  }
}
</script>

<style lang="less" scoped>
.ace-div {
  width: 350px;
  height: 280px;
}
</style>
