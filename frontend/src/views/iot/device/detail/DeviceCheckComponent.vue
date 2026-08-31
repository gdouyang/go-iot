<template>
  <el-popover width="auto" trigger="click" placement="bottom">
    <el-descriptions border :column="1" size="small" class="check-info" v-loading="loading">
      <el-descriptions-item label="设备激活" v-if="checkInfo.deviceActive !== null">
        <el-tag :type="checkInfo.deviceActive ? 'success' : 'danger'">{{
          displayText(checkInfo.deviceActive)
        }}</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="产品发布" v-if="checkInfo.productActive !== null">
        <el-tag :type="checkInfo.productActive ? 'success' : 'danger'">{{
          displayText(checkInfo.productActive)
        }}</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="编辑码脚本" v-if="checkInfo.scriptActive !== null">
        <el-tag :type="checkInfo.scriptActive ? 'success' : 'danger'">{{
          displayText(checkInfo.scriptActive)
        }}</el-tag>
      </el-descriptions-item>
      <el-descriptions-item label="网络服务" v-if="checkInfo.serverActive !== null">
        <el-tag :type="checkInfo.serverActive ? 'success' : 'danger'">{{
          displayText(checkInfo.serverActive)
        }}</el-tag>
      </el-descriptions-item>
    </el-descriptions>
    <template #reference>
      <el-button link type="success" class="link" @click="doConnectionCheck">诊断</el-button>
    </template>
  </el-popover>
</template>

<script lang="jsx">
import { connectionCheck } from '../api.js'

export default {
  name: 'DeviceCheckComponent',
  components: {},
  props: {
    deviceId: {
      type: String,
      default: ''
    }
  },
  data() {
    return {
      loading: false,
      checkInfo: {}
    }
  },
  computed: {},
  created() {},
  mounted() {},
  unmounted() {},
  methods: {
    doConnectionCheck() {
      this.loading = true
      connectionCheck(this.deviceId)
        .then((data) => {
          if (data.success) {
            this.checkInfo = data.result
          }
        })
        .finally(() => {
          this.loading = false
        })
    },
    displayText(value) {
      return value ? '通过' : '未通过'
    }
  }
}
</script>

<style lang="less" scoped>
.check-info {
  :deep(.el-descriptions__label) {
    min-width: 100px;
  }
  :deep(.el-descriptions__content) {
    min-width: 100px;
  }
}
</style>
