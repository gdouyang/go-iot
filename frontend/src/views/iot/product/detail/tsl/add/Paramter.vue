<template>
  <el-drawer
    title="新增参数"
    placement="right"
    :model-value="true"
    :close-on-click-modal="false"
    width="30%"
    class="footer-drawer"
  >
    <el-form :model="formData" ref="form" label-width="auto">
      <el-form-item
        label="参数标识"
        prop="id"
        :rules="[
          { required: true, message: '请输入参数标识' },
          { max: 32, message: '参数标识不超过32个字符' },
          {
            pattern: new RegExp(/^[0-9a-zA-Z_\-]+$/, 'g'),
            message: '参数标识只能由数字、字母、下划线、中划线组成'
          }
        ]"
      >
        <el-input v-model="formData.id" placeholder="请输入参数标识" :disabled="isEdit" />
      </el-form-item>
      <el-form-item
        label="参数名称"
        prop="name"
        :rules="[
          { required: true, message: '请输入参数名称' },
          { max: 200, message: '参数名称不超过200个字符' }
        ]"
      >
        <el-input v-model="formData.name" placeholder="请输入参数名称" />
      </el-form-item>
      <DataTypeItemSimple
        label="数据类型"
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
</template>

<script lang="jsx">
// import _ from 'lodash-es'
import DataTypeItemSimple from '../components/DataTypeItemSimple.vue'
import { getPropertiesData } from '../components/data.js'
export default {
  name: 'Paramter',
  components: {
    DataTypeItemSimple
  },
  props: {
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      formData: {},
      isEdit: false
    }
  },
  created() {
    this.formData = getPropertiesData(this.data)
    if (this.data && this.data.id) {
      this.isEdit = true
    }
  },
  mounted() {},
  methods: {
    saveData() {
      this.$refs.form.validate((valid) => {
        if (valid) {
          this.$emit('save', this.formData)
        }
      })
    }
  }
}
</script>
