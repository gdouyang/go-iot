<template>
  <div>
    <el-form-item
      label="标题"
      prop="template.title"
      :rules="[
        { required: true, message: '请输入标题' },
        { max: 255, message: '长度不超过255' }
      ]"
    >
      <el-input v-model="data.template.title" />
    </el-form-item>
    <el-form-item
      label="正文"
      prop="template.text"
      :rules="[
        { required: true, message: '请输入正文' },
        { max: 5120, message: '长度不超过5120' }
      ]"
    >
      <MdEditor v-model="data.template.text" :toolbarsExclude="toolbars" />
    </el-form-item>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { MdEditor } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'
export default {
  name: 'Dingtalk',
  components: {
    MdEditor
  },
  props: {
    data: {
      type: Object,
      default: null
    }
  },
  data() {
    return {
      template: {},
      toolbars: [
        'image',
        'save',
        'katex',
        'previewOnly',
        'catalog',
        'mermaid',
        'pageFullscreen',
        'htmlPreview',
        'github'
      ]
    }
  },
  created() {
    let str = this.data.template
    if (_.isObject(str)) {
      str = JSON.stringify(str)
    }
    if (_.isNil(str) || _.isEmpty(str)) {
      str = '{"title":"", "text":""}'
    }
    const template = JSON.parse(str)
    this.data.template = template
  },
  methods: {
    getTemplate() {
      return JSON.stringify(this.data.template)
    }
  }
}
</script>

<style lang="less"></style>
