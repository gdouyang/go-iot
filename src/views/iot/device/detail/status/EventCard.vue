<template>
  <ChartCard :title="item.name">
    <div>{{ formatValue }}次</div>
    <span>
      <el-badge v-if="item.expands.level === 'ordinary'" status="processing" text="普通" />
      <el-badge v-if="item.expands.level === 'warn'" status="warning" text="警告" />
      <el-badge v-if="item.expands.level === 'urgent'" status="error" text="紧急" />
    </span>
  </ChartCard>
</template>

<script lang="jsx">
// import _ from 'lodash-es'
import ChartCard from './ChartCard.vue'
import { queryEvent } from '../../api.js'
export default {
  name: 'EventCard',
  components: {
    ChartCard
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
          formatValue: null,
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
      formatValue: 0
    }
  },
  mounted() {
    this.getValue()
  },
  methods: {
    inc() {
      this.formatValue++
    },
    getValue() {
      queryEvent(this.device.id, this.item.id, { pageNum: 1, pageSize: 1 }).then((resp) => {
        if (resp.success) {
          this.formatValue = resp.result.totalCount
        }
      })
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
</style>
