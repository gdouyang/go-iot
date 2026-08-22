<template>
  <Echart :options="lineOptionsData" v-bind="$attrs" />
</template>

<script lang="jsx">
export default {
  name: 'Chart',
  components: {},
  props: {
    data: {
      type: Object,
      default: () => {
        return []
      }
    }
  },
  data() {
    return {
      lineOptionsData: {
        tooltip: {
          trigger: 'axis',
          axisPointer: {
            type: 'cross'
          },
          padding: [5, 10]
        },
        grid: {
          top: '1%',
          left: '1%',
          right: '1%',
          bottom: '0%',
          containLabel: false
        },
        xAxis: {
          type: 'category',
          boundaryGap: false,
          show: false,
          data: []
        },
        yAxis: {
          show: false,
          axisTick: {
            show: false
          }
        },
        series: [
          {
            type: 'line',
            areaStyle: {},
            data: []
          }
        ]
      }
    }
  },
  watch: {
    data: {
      handler(newVal) {
        this.setData(newVal)
      },
      deep: true
    }
  },
  mounted() {
    this.setData(this.data)
  },
  methods: {
    setData(data) {
      this.lineOptionsData.xAxis.data = data.map((item) => {
        return item.x
      })
      this.lineOptionsData.series[0].data = data.map((item) => {
        return item.y
      })
    }
  }
}
</script>

<style lang="less" scoped></style>
