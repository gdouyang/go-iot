<template>
  <div class="tsl-index-wrap" v-loading="loading">
    <el-alert type="warning" :closable="false" show-icon class="mb-3">
      <template #title>
        <div class="font-bold text-14px"
          >物模型是设备数据接入、时序存储与规则流转的底层基准，请严谨变更</div
        >
      </template>
      <div class="text-12px leading-relaxed text-[var(--el-text-color-regular)] mt-1.5 space-y-1">
        <div>
          1.
          <b>时序存储联动</b
          >：系统会自动为物模型属性与事件在底层时序数据库创建对应的数据表与字段映射，<b>变更标识（ID）或数据类型将导致时序数据入库失败或历史图表异常</b>。
        </div>
        <div>
          2. <b>编解码脚本绑定</b>：通信脚本（<code>SaveProperties</code> /
          <code>SaveEvents</code
          >）深度依赖物模型标识，<b>改动标识会导致设备上报报文无法匹配而丢包</b>。
        </div>
        <div>
          3.
          <b>规则与告警联动</b
          >：已生效的自动化规则和告警触发器均以物模型标识为判断条件，<b>删除/修改字段将导致关联告警失效</b>。
        </div>
        <div>
          4.
          <b>生产变更规范</b
          >：对已有在线设备的产品，<b>切勿随意修改或删除已有属性</b>；建议采用新增属性或升级新产品版本的方式平滑迭代。
        </div>
        <div>
          5.
          <b>生效必做操作</b
          >：保存修改后，<b>必须在产品详情页点击【重新应用配置】</b>，底层时序库结构与通信节点才会同步热重载生效。
        </div>
      </div>
    </el-alert>
    <div class="relative">
      <div style="display: inline-block; position: absolute; right: 0; top: 4px; z-index: 10">
        <el-button @click="importTSL" style="margin-right: 5px">导入物模型</el-button>
        <el-button @click="showTSL">物模型</el-button>
      </div>
      <el-tabs model-value="1">
        <el-tab-pane label="属性定义" name="1">
          <Properties :product="product" :data="propertyData" @save="saveProperties" />
        </el-tab-pane>
        <el-tab-pane label="功能定义" name="2">
          <Functions :product="product" :data="functionsData" @save="saveFunctions" />
        </el-tab-pane>
        <el-tab-pane label="事件定义" name="3">
          <Events :data="eventsData" @save="saveEvents" />
        </el-tab-pane>
      </el-tabs>
    </div>
    <TslImportDialog ref="TslImportDialog" @import="saveAll" />
  </div>
</template>

<script lang="jsx">
// import _ from 'lodash-es'
import Properties from './tsl/Properties.vue'
import Functions from './tsl/Functions.vue'
import Events from './tsl/Events.vue'
import TslImportDialog from './tsl/TslImportDialog.vue'
export default {
  name: 'TSL',
  components: {
    Properties,
    Functions,
    Events,
    TslImportDialog
  },
  props: {
    product: {
      type: Object,
      default: () => {}
    },
    propertyData: {
      type: [Object, Array],
      default: () => null
    },
    functionsData: {
      type: [Object, Array],
      default: () => null
    },
    eventsData: {
      type: [Object, Array],
      default: () => null
    }
  },
  data() {
    return {
      properties: [],
      loading: false
    }
  },
  mounted() {},
  methods: {
    saveProperties(data, onlySave) {
      this.$emit('save', 'properties', data, onlySave)
    },
    saveFunctions(data, onlySave) {
      this.$emit('save', 'function', data, onlySave)
    },
    saveEvents(data, onlySave) {
      this.$emit('save', 'event', data, onlySave)
    },
    saveAll(data, onlySave) {
      this.$emit('save', 'all', data, onlySave)
    },
    importTSL() {
      this.$refs.TslImportDialog.open(null, true)
    },
    showTSL() {
      this.$refs.TslImportDialog.open(this.product.metadata)
    }
  }
}
</script>

<style lang="less" scoped></style>
