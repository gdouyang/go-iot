<template>
  <el-drawer
    title="修改配置"
    width="500"
    :model-value="true"
    :close-on-click-modal="false"
    @close="visibleChange(false)"
  >
    <el-form ref="addFormRef" :model="configuration" label-width="auto">
      <el-row :gutter="16">
        <el-col>
          <el-form-item
            prop="property"
            label="Key"
            :rules="[{ required: true, message: 'Key不能为空', trigger: 'blur' }]"
          >
            <el-input v-model="configuration.property" :disabled="true" :maxlength="32" />
          </el-form-item>
          <el-form-item label="值">
            <el-input
              v-if="configuration.type === 'password'"
              v-model="configuration.value"
              type="password"
              show-password
              :maxlength="100"
            />
            <el-input v-model="configuration.value" :maxlength="100" v-else />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>
    <div class="drawer-footer">
      <el-button style="margin-right: 8px" @click="visibleChange(false)">关闭</el-button>
      <el-button type="primary" @click="saveData">保存</el-button>
    </div>
  </el-drawer>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { updateDevice } from '@/views/iot/device/api.js'
const defaultData = { property: '', desc: '', type: null, value: '' }
export default {
  name: 'ConfigurationAdd',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    deviceId: {
      type: String,
      default: () => null
    },
    allConfig: {
      type: Object,
      default: () => {}
    },
    data: {
      type: Object,
      default: () => null
    }
  },
  data() {
    return {
      configuration: _.cloneDeep(defaultData),
      isEdit: false
    }
  },
  mounted() {
    this.visibleChange(true)
  },
  methods: {
    visibleChange(flag) {
      if (!flag) {
        this.$emit('close')
      } else {
        const d = _.cloneDeep(this.data)
        this.configuration = d
      }
    },
    saveData() {
      this.$refs.addFormRef.validate((valid, object) => {
        if (valid) {
          const p = _.cloneDeep(this.configuration)
          this.updateData(p)
        }
      })
    },
    updateData(item) {
      const p = _.cloneDeep(this.allConfig)
      p[item.property] = item.value
      const param = {
        id: this.deviceId,
        metaconfig: p
      }
      this.updateVisible = false
      updateDevice(param).then((resp) => {
        if (resp.success) {
          this.$message.success('配置信息修改成功')
          this.visibleChange(false)
          this.$emit('refresh')
        }
      })
    }
  }
}
</script>

<style lang="less" scoped></style>
