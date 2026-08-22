<template>
  <el-row :gutter="24" v-loading="loading">
    <el-col :xs="12" :sm="10" :md="8" :lg="6" :xl="6" style="margin-bottom: 24px">
      <DeviceState :state="device.state" :device="device" />
    </el-col>
    <el-col
      :xs="12"
      :sm="10"
      :md="8"
      :lg="6"
      :xl="6"
      style="margin-bottom: 24px"
      v-for="item in properties"
      :key="item.id"
    >
      <PropertiesCard :item="item" :device="device" :ref="'propCard' + item.id" />
    </el-col>
    <el-col
      :xs="12"
      :sm="10"
      :md="8"
      :lg="6"
      :xl="6"
      style="margin-bottom: 24px"
      v-for="item in events"
      :key="item.id"
    >
      <EventCard :item="item" :device="device" :ref="'eventCard' + item.id" />
    </el-col>
  </el-row>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { queryProperty } from '@/views/iot/device/api.js'
import DeviceState from './status/DeviceState.vue'
import PropertiesCard from './status/PropertiesCard.vue'
import EventCard from './status/EventCard.vue'
export default {
  name: 'DeviceStatus',
  components: {
    DeviceState,
    PropertiesCard,
    EventCard
  },
  props: {
    device: {
      type: Object,
      default: () => {}
    },
    realtimeData: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      properties: [],
      events: [],
      loading: true
    }
  },
  watch: {
    realtimeData(newVal) {
      if (newVal.type === 'property') {
        const propData = newVal.data
        const that = this
        _.forEach(this.properties, (prop) => {
          if (!_.isNil(propData[prop.id])) {
            prop.list.push({
              timeString: newVal.createTime,
              value: propData[prop.id]
            })
            if (_.size(prop.list) > 20) {
              var list = []
              for (var i = prop.list.length - 20; i < prop.list.length; i++) {
                list.push(prop.list[i])
              }
              prop.list = list
            }
            prop.value = propData[prop.id]
            that.$refs['propCard' + prop.id][0].getValue()
          }
        })
      } else if (newVal.type === 'event') {
        this.$refs['eventCard' + newVal.eventId][0].inc()
      }
    }
  },
  mounted() {
    this.propertiesRealTime()
  },
  beforeUnmount() {},
  methods: {
    propertiesRealTime() {
      const device = this.device
      this.loading = true
      queryProperty(device.id, {})
        .then((resp) => {
          const metadata = JSON.parse(this.device.metadata)
          const properties = _.cloneDeep(metadata.properties)
          this.events = metadata.events
          _.forEach(properties, (prop) => {
            const list = []
            _.forEach(resp.result.list, (item) => {
              if (!_.isNil(item[prop.id])) {
                list.push({
                  timeString: item.createTime,
                  value: item[prop.id]
                })
              }
            })
            prop.list = list.reverse()
          })
          this.properties = properties
        })
        .finally(() => {
          this.loading = false
        })
    }
  }
}
</script>

<style lang="less" scoped></style>
