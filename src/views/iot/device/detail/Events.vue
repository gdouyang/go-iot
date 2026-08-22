<template>
  <div>
    <ContentWrap>
      <!-- 物模型未配置任何事件定义 -->
      <el-empty v-if="!events.length" description="暂未配置设备事件" :image-size="120" />
      <template v-else>
        <div>
          <el-form label-width="auto">
            <el-row :gutter="24">
              <el-col :md="6" :sm="24">
                <el-form-item label="事件">
                  <el-select v-model="eventId" @change="onEventIdChange">
                    <el-option
                      v-for="(item, index) in events"
                      :key="index"
                      :value="item.id"
                      :label="item.name"
                    />
                  </el-select>
                </el-form-item>
              </el-col>
              <el-col :md="8" :sm="24">
                <el-form-item label="日期">
                  <el-date-picker
                    v-model="searchParams.createTime"
                    type="datetimerange"
                    value-format="YYYY-MM-DD HH:mm:ss"
                    format="YYYY-MM-DD HH:mm:ss"
                    start-placeholder="开始时间"
                    end-placeholder="结束时间"
                    :default-time="defaultTime"
                    :shortcuts="dateShortcuts"
                    style="width: 100%"
                  />
                </el-form-item>
              </el-col>
              <el-col :md="6" :sm="24">
                <div :style="{ overflow: 'hidden' }">
                  <div :style="{ marginLeft: '10px' }">
                    <el-button type="primary" @click="search"> 查询 </el-button>
                    <el-button :style="{ marginLeft: '8px' }" @click="resetSearch">
                      重置
                    </el-button>
                  </div>
                </div>
              </el-col>
            </el-row>
          </el-form>
        </div>
        <!-- 始终渲染表格：有配置时即使查询无数据也显示空表 -->
        <PageTable
          v-if="tableUrl"
          ref="tb"
          :key="tableUrl"
          :columns="columns"
          :url="tableUrl"
          emptyText="暂无事件数据"
        />
      </template>
    </ContentWrap>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import dayjs from 'dayjs'
import { getDeviceEventsUrl } from '@/views/iot/device/api.js'
import { getDefaultCreateTimeRange, defaultTime, dateShortcuts } from './dateRange.js'

export default {
  name: 'Events',
  props: {
    device: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      tableUrl: '',
      columns: [],
      defaultTime,
      dateShortcuts,
      searchParams: {
        createTime: getDefaultCreateTimeRange()
      },
      events: [],
      eventId: ''
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    init() {
      let events = []
      try {
        const metadata = this.device?.metadata
        if (metadata) {
          const parsed = typeof metadata === 'string' ? JSON.parse(metadata) : metadata
          events = parsed?.events || []
        }
      } catch (e) {
        console.error('解析设备物模型失败', e)
        events = []
      }
      this.events = Array.isArray(events) ? events : []
      if (!this.events.length) {
        this.tableUrl = ''
        this.columns = []
        return
      }
      this.eventId = this.events[0].id
      this.onEventIdChange()
    },
    onEventIdChange() {
      if (!this.eventId) {
        this.tableUrl = ''
        return
      }
      this.tableUrl = getDeviceEventsUrl(this.device.id, this.eventId)
      const event = _.find(this.events, (ev) => ev.id === this.eventId)
      const columns = []
      if (event) {
        if (event.type === 'object') {
          _.forEach(event.properties, (prop) => {
            columns.push({ field: prop.id, label: prop.name })
          })
        } else {
          columns.push({ field: event.id, label: event.name })
        }
      }
      columns.push({ field: 'createTime', label: '时间' })
      this.columns = columns
      // 切换事件时保留当前日期条件；首次进入用默认最近 1 个月
      if (_.isEmpty(this.searchParams.createTime)) {
        this.searchParams.createTime = getDefaultCreateTimeRange()
      }
      this.$nextTick(() => {
        this.search()
      })
    },
    search() {
      if (!this.$refs.tb) {
        return
      }
      const params = []
      if (!_.isEmpty(this.searchParams.createTime) && this.searchParams.createTime.length === 2) {
        const formatDate = this.searchParams.createTime.map((e) =>
          dayjs(e).format('YYYY-MM-DD HH:mm:ss')
        )
        params.push({ key: 'createTime', oper: 'BTW', value: formatDate.join(',') })
      }
      this.$refs.tb.search(params)
    },
    resetSearch() {
      this.searchParams = {
        createTime: getDefaultCreateTimeRange()
      }
      this.search()
    }
  }
}
</script>

<style lang="less" scoped></style>
