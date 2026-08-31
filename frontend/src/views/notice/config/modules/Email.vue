<template>
  <div>
    <el-form-item
      label="标题"
      prop="template.subject"
      :rules="[{ max: 255, message: '长度不超过255' }]"
    >
      <el-input v-model="data.template.subject" />
    </el-form-item>
    <el-form-item
      label="正文"
      prop="template.text"
      :rules="[
        { required: true, message: '请输入正文' },
        { max: 5120, message: '长度不超过5120' }
      ]"
    >
      <el-textarea :rows="8" placeholder="html" v-model="data.template.text" :max-length="60000" />
    </el-form-item>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
export default {
  name: 'Email',
  components: {},
  props: {
    data: {
      type: Object,
      default: null
    }
  },
  data() {
    return {
      template: {}
    }
  },
  created() {
    let str = this.data.template
    if (_.isObject(str)) {
      str = JSON.stringify(str)
    }
    if (_.isNil(str) || _.isEmpty(str)) {
      str = '{"subject":"", "text":""}'
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
