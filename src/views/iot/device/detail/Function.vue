<template>
  <el-card shadow="never" title="功能调试" v-loading="spinning">
    <el-empty v-if="!functionsSelectList.length" description="暂未配置设备功能" :image-size="120" />
    <el-collapse v-else v-model="activeKey" style="width: 500px">
      <el-collapse-item v-for="f in functionsSelectList" :key="f.id" :name="f.name" :title="f.name">
        <div style="text-align: right">
          <el-button link type="primary" @click.prevent="debugFunction(f)"> 发送指令 </el-button>
        </div>
        <el-empty
          v-if="!f.inputs || f.inputs.length < 1"
          description="该功能无输入参数"
          :image-size="80"
        />
        <p v-else>
          <FunctionForm :inputs="f.inputs" :ref="'funcForm-' + f.id" />
        </p>
      </el-collapse-item>
    </el-collapse>
  </el-card>
</template>

<script lang="jsx">
import { cmdInvoke } from '@/views/iot/device/api.js'
import FunctionForm from './functions/FunctionForm.vue'
export default {
  name: 'DeviceFunction',
  components: {
    FunctionForm
  },
  props: {
    device: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      functionsSelectList: [],
      activeKey: [],
      spinning: false
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    init() {
      let functions = []
      try {
        const metadata = this.device?.metadata
        if (metadata) {
          const parsed = typeof metadata === 'string' ? JSON.parse(metadata) : metadata
          functions = parsed?.functions || []
        }
      } catch (e) {
        console.error('解析设备物模型失败', e)
        functions = []
      }
      this.functionsSelectList = Array.isArray(functions) ? functions : []
    },
    debugFunction(fun) {
      const functionId = fun.id
      const deviceId = this.device.id
      const refId = 'funcForm-' + functionId
      const ref = this.$refs[refId]
      const params = {
        functionId: functionId
      }
      if (ref) {
        params.data = ref[0].getData()
      }
      this.spinning = true
      params.offlineCache = true
      cmdInvoke(deviceId, params)
        .then((resp) => {
          if (resp.success) {
            this.$message.success(resp.message || '操作成功')
          }
        })
        .finally(() => {
          this.spinning = false
        })
    }
  }
}
</script>

<style lang="less" scoped>
// :deep(.el-collapse-item__header) {
//   background-color: #fafafa;
// }
</style>
