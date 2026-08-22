<template>
  <div>
    <el-drawer
      title="编辑事件定义"
      placement="right"
      :model-value="true"
      :close-on-click-modal="false"
      width="30%"
      class="footer-drawer"
    >
      <el-form :model="formData" ref="form" label-width="auto">
        <el-form-item
          label="事件标识"
          prop="id"
          :rules="[
            { required: true, message: '请输入事件标识' },
            { max: 32, message: '事件标识不超过32个字符' },
            {
              pattern: new RegExp(/^[0-9a-zA-Z_\-]+$/, 'g'),
              message: '事件标识只能由数字、字母、下划线、中划线组成'
            }
          ]"
        >
          <el-input v-model="formData.id" placeholder="请输入事件标识" :disabled="isEdit" />
        </el-form-item>
        <el-form-item
          label="事件名称"
          prop="name"
          :rules="[
            { required: true, message: '请输入事件名称' },
            { max: 200, message: '事件名称不超过200个字符' }
          ]"
        >
          <el-input v-model="formData.name" placeholder="请输入事件名称" />
        </el-form-item>
        <el-form-item
          label="事件级别"
          prop="expands.level"
          :rules="[{ required: true, message: '请选择' }]"
        >
          <el-radio-group v-model="formData.expands.level">
            <el-radio value="ordinary">普通</el-radio>
            <el-radio value="warn">警告</el-radio>
            <el-radio value="urgent">紧急</el-radio>
          </el-radio-group>
        </el-form-item>
        <!-- -->
        <DataTypeItem
          label="输出参数"
          :data="formData"
          :rules="[{ required: true, message: '请选择' }]"
        />
        <!-- -->
        <el-form-item label="描述" prop="description">
          <el-textarea v-model="formData.description" :rows="3" />
        </el-form-item>
      </el-form>
      <div class="drawer-footer">
        <el-button style="margin-right: 8px" @click="$emit('close')">关闭</el-button>
        <el-button type="primary" @click="saveData">保存</el-button>
      </div>
    </el-drawer>
    <Paramter
      v-if="outputVisible"
      :data="currentParameter"
      @close="closeOutput"
      @save="saveOutput"
    />
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import DataTypeItem from '../components/DataTypeItem.vue'
import Paramter from '../add/Paramter.vue'

import { getEventsData } from '../components/data.js'
const defaultFormData = getEventsData()
export default {
  name: 'EventsAdd',
  components: {
    DataTypeItem,
    Paramter
  },
  props: {
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      formData: _.assign({}, defaultFormData),
      isEdit: false,
      outputVisible: false
    }
  },
  watch: {},
  created() {
    this.formData = getEventsData(this.data)
    if (this.data && this.data.id) {
      this.isEdit = true
    }
  },
  mounted() {},
  methods: {
    saveData() {
      this.$refs.form.validate((valid) => {
        if (valid) {
          if (this.formData.type === 'enum') {
            const elements = (this.formData.elements || []).filter(
              (item) => item && item.value !== '' && item.value !== null && item.value !== undefined
            )
            if (!elements.length) {
              this.$message.error('枚举类型必须至少配置一个有效的枚举项标识(value)')
              return
            }
            this.formData.elements = elements
          } else {
            delete this.formData.elements
          }
          if (this.formData.type === 'object') {
            if (!this.formData.properties || !this.formData.properties.length) {
              this.$message.error('结构体类型(object)必须至少添加一个参数(properties)')
              return
            }
          } else {
            delete this.formData.properties
          }
          if (this.formData.type !== 'array') {
            delete this.formData.elementType
          }
          this.$emit('save', this.formData)
        }
      })
    },
    saveOutput(item) {
      this.data.properties = item
      this.closeOutput()
    },
    closeOutput() {
      this.outputVisible = false
    }
  }
}
</script>

<style lang="less" scoped></style>
