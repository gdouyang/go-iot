<template>
  <ChartCard title="设备状态">
    <template #total>
      <span>{{ deviceStateText }}</span>
    </template>
    <span v-if="state === 'online'">上线时间：{{ time }}</span>
    <span v-if="state === 'offline'">离线时间：{{ time }}</span>
  </ChartCard>
</template>

<script lang="jsx">
import ChartCard from './ChartCard.vue'
import { getStatusText, queryLogs } from '../../api.js'
// eslint-disable-next-line no-unused-vars
// import dayjs from 'dayjs'
export default {
  name: 'DeviceState',
  components: {
    ChartCard
  },
  props: {
    state: {
      type: String,
      default: () => ''
    },
    device: {
      type: Object,
      default: () => {
        return {}
      }
    }
  },
  data() {
    return {
      time: ''
    }
  },
  computed: {
    deviceStateText() {
      const status = this.state
      return getStatusText(status)
    }
  },
  mounted() {
    this.GetTime()
  },
  methods: {
    GetTime() {
      if (this.state === 'online' || this.state === 'offline') {
        queryLogs(this.device.id, {
          pageSize: 1,
          condition: [{ key: 'type', value: this.state }]
        }).then((resp) => {
          if (resp.success && resp.result.list.length > 0) {
            this.time = resp.result.list[0].createTime
          }
        })
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
</style>
