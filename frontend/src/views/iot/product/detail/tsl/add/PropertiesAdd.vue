<template>
  <el-drawer
    title="编辑属性"
    placement="right"
    :model-value="true"
    :close-on-click-modal="false"
    width="30%"
    class="footer-drawer"
  >
    <el-form :model="formData" ref="form" label-width="auto">
      <el-form-item
        label="属性标识"
        prop="id"
        :rules="[
          { required: true, message: '请输入属性标识' },
          { max: 32, message: '属性标识不超过32个字符' },
          {
            pattern: new RegExp(/^[0-9a-zA-Z_\-]+$/, 'g'),
            message: '属性标识只能由数字、字母、下划线、中划线组成'
          }
        ]"
      >
        <el-input v-model="formData.id" placeholder="请输入属性标识" :disabled="isEdit" />
      </el-form-item>
      <el-form-item
        label="属性名称"
        prop="name"
        :rules="[
          { required: true, message: '请输入属性名称' },
          { max: 200, message: '属性名称不超过200个字符' }
        ]"
      >
        <el-input v-model="formData.name" placeholder="请输入属性名称" />
      </el-form-item>
      <!-- -->
      <DataTypeItem
        label="数据类型"
        v-model:data="formData"
        :rules="[{ required: true, message: '请选择' }]"
      />
      <!-- -->
      <!-- <el-form-item
        label="是否只读"
        prop="expands.readOnly"
        :rules="[ { required: true, message: '请选择' } ]"
      >
        <el-radio-group v-model="formData.expands.readOnly">
          <el-radio value="true">是</el-radio>
          <el-radio value="false">否</el-radio>
        </el-radio-group>
      </el-form-item> -->
      <el-form-item label="描述" prop="description">
        <el-textarea v-model="formData.description" :rows="3" />
      </el-form-item>
    </el-form>
    <div class="drawer-footer">
      <el-button style="margin-right: 8px" @click="$emit('close')">关闭</el-button>
      <el-button type="primary" @click="saveData">保存</el-button>
    </div>
  </el-drawer>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import DataTypeItem from '../components/DataTypeItem.vue'

import { getPropertiesData } from '../components/data.js'
const defaultFormData = getPropertiesData()
export default {
  name: 'PropertiesAdd',
  components: {
    DataTypeItem
  },
  props: {
    product: {
      type: Object,
      default: () => {}
    },
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      formData: _.cloneDeep(defaultFormData),
      isEdit: false
    }
  },
  watch: {},
  created() {
    this.formData = _.cloneDeep(_.assign({}, defaultFormData, this.data))
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
    }
  }
}
</script>

<style lang="less" scoped></style>
