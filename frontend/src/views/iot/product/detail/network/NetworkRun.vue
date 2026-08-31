<template>
  <span v-if="!isNetClientType">
    <el-badge
      :color="isRuning ? 'green' : 'red'"
      :text="isRuning ? '运行中' : '停止'"
      style="margin-left: 10px"
    />
    <el-popconfirm
      title="确认启动网络服务？"
      width="200px"
      @confirm="runNetwork('start')"
      v-if="!isRuning"
    >
      <template #reference>
        <el-button link type="primary" size="small">启动网络服务</el-button>
      </template>
    </el-popconfirm>
    <el-popconfirm
      title="确认停止网络服务？"
      width="200px"
      @confirm="runNetwork('stop')"
      v-if="isRuning"
    >
      <template #reference>
        <el-button link type="primary" size="small">停止网络服务</el-button>
      </template>
    </el-popconfirm>
  </span>
</template>

<script lang="jsx">
import { runNetwork } from '@/views/iot/product/api.js'
export default {
  name: 'NetworkRun',
  components: {},
  props: {
    productId: {
      type: String,
      default: null
    },
    network: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {}
  },
  computed: {
    isRuning() {
      return this.network.state === 'runing'
    },
    isNetClientType() {
      return this.network.type === 'TCP_CLIENT' || this.network.type === 'MQTT_CLIENT'
    }
  },
  created() {},
  methods: {
    runNetwork(state) {
      return runNetwork(this.productId, state).then((resp) => {
        if (resp.success) {
          this.$message.success('操作成功')
          this.$emit('success')
        }
        return resp
      })
    }
  }
}
</script>

<style lang="less" scoped></style>
