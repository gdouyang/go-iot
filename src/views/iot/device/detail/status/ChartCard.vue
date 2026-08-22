<template>
  <el-card shadow="never">
    <div
      ><el-text>{{ title }}</el-text></div
    >
    <div class="total">
      <slot name="total"></slot>
    </div>
    <div class="chart-card-content">
      <slot></slot>
    </div>
  </el-card>
</template>

<script lang="jsx">
// import _ from 'lodash-es'
export default {
  name: 'ChartCard',
  components: {},
  props: {
    title: {
      type: String,
      default: ''
    }
  },
  data() {
    return {}
  },
  mounted() {},
  methods: {}
}
</script>

<style lang="less" scoped>
.total {
  overflow: hidden;
  text-overflow: ellipsis;
  word-break: break-all;
  white-space: nowrap;
  margin-top: 4px;
  margin-bottom: 0;
  font-size: 30px;
  line-height: 38px;
  height: 38px;
}
.chart-card-content {
  position: relative;
  height: 46px;
  width: 100%;
  display: flex;
  align-items: flex-end;
}
</style>
