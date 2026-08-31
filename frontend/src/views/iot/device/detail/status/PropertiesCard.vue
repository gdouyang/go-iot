<template>
  <ChartCard :title="item.name">
    <template #total>
      <div class="prop-data" :title="lastData">
        {{ lastData }}<span class="unit">{{ unit }}</span>
      </div>
    </template>
    <Chart :data="data.visitData" :height="50" />
  </ChartCard>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import ChartCard from './ChartCard.vue'
import Chart from './Chart.vue'
export default {
  name: 'PropertiesCard',
  components: {
    ChartCard,
    Chart
  },
  props: {
    item: {
      type: Object,
      default: () => {
        return {
          name: '',
          expands: {
            readOnly: null
          },
          id: '',
          type: '',
          unit: '',
          list: [],
          value: null,
          visitData: null
        }
      }
    },
    device: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      data: {
        visitData: []
      },
      lastData: '',
      unit: ''
    }
  },
  created() {
    this.getValue()
  },
  methods: {
    getValue() {
      const item = _.assign(this.data, this.item)
      if (item && !_.isEmpty(item.list)) {
        const length = item.list.length
        const value = item.list[length - 1]
        const dataType = typeof value.value
        if (dataType === 'object') {
          item.value = JSON.stringify(value.value) || '/'
        } else {
          item.value = _.isNil(value.value) ? '/' : value.value
        }

        // 特殊类型
        const valueType = item.type
        if (
          valueType === 'int' ||
          valueType === 'float' ||
          valueType === 'double' ||
          valueType === 'long'
        ) {
          const visitData = []
          item.list.forEach((data) => {
            visitData.push({
              x: data.timeString,
              y: Math.floor(Number(data.value) * 100) / 100
            })
          })
          item.visitData = visitData
        }
      }
      this.data = item
      this.getLastData()
    },
    getLastData() {
      if (_.isNil(this.data.value)) {
        this.lastData = '/'
        return
      }
      const unit = _.toString(this.item.unit)
      if (typeof this.data.value === 'object') {
        this.unit = ''
        this.lastData = JSON.stringify(this.data.value)
      } else {
        this.unit = unit
        this.lastData = this.data.value
      }
    }
  }
}
</script>

<style lang="less" scoped>
.card-value {
  color: rgba(0, 0, 0, 0.85);
  font-size: 30px;
  overflow: hidden;
  line-height: 38px;
  white-space: nowrap;
  text-overflow: ellipsis;
  word-break: break-all;
}
.prop-data {
  overflow: hidden;
  text-overflow: ellipsis;
  .unit {
    font-size: 14px;
  }
}
</style>
