<template>
  <div>
    <ContentWrap>
      <div>
        <el-form label-width="auto">
          <el-row :gutter="24">
            <el-col :md="5" :sm="24">
              <el-form-item label="日志类型">
                <el-select v-model="searchParams.type" multiple clearable>
                  <el-option
                    v-for="(item, index) in selectOptions"
                    :key="index"
                    :value="item.id"
                    :label="item.name"
                  />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :md="5" :sm="24">
              <el-form-item label="TraceId">
                <el-input v-model="searchParams.traceId" maxlength="100" clearable />
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
                  <el-button @click="resetSearch"> 重置 </el-button>
                </div>
              </div>
            </el-col>
          </el-row>
        </el-form>
      </div>
      <PageTable ref="tb" :columns="columns" :url="tableUrl" emptyText="暂无日志数据" />
    </ContentWrap>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import dayjs from 'dayjs'
import { getDeviceLogsUrl } from '@/views/iot/device/api.js'
import { getDefaultCreateTimeRange, defaultTime, dateShortcuts } from './dateRange.js'

const defaultOptions = [
  // { id: 'event', name: '事件上报' },
  // { id: 'readProperty', name: '属性读取' },
  // { id: 'writeProperty', name: '属性修改' },
  // { id: 'reportProperty', name: '属性上报' },
  { id: 'call', name: '调用' },
  { id: 'reply', name: '回复' },
  { id: 'offline', name: '下线' },
  { id: 'online', name: '上线' }
  // { id: 'other', name: '其它' }
]
let typeMap = {}
defaultOptions.forEach((item) => {
  typeMap[item.id] = item.name
})

export default {
  name: 'DeviceLog',
  props: {
    deviceId: {
      type: String,
      default: null
    }
  },
  data() {
    return {
      tableUrl: '',
      selectOptions: defaultOptions,
      defaultTime,
      dateShortcuts,
      columns: [
        {
          field: 'type',
          label: '类型',
          width: '120px',
          slots: { default: (data) => typeMap[data.row.type] || '-' }
        },
        { field: 'traceId', label: 'TraceId', width: '200px' },
        { field: 'createTime', label: '时间', width: '200px' },
        { field: 'content', label: '内容' },
        {
          label: '操作',
          field: 'action',
          width: '120px',
          slots: {
            default: (data) => {
              return (
                <el-button link type="primary" onClick={() => this.showDetail(data.row)}>
                  查看
                </el-button>
              )
            }
          }
        }
      ],
      searchParams: {
        type: [],
        createTime: getDefaultCreateTimeRange(),
        traceId: ''
      }
    }
  },
  created() {
    this.tableUrl = getDeviceLogsUrl(this.deviceId)
  },
  mounted() {
    this.$nextTick(() => {
      this.search()
    })
  },
  methods: {
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
      if (!_.isEmpty(this.searchParams.type)) {
        params.push({ key: 'type', oper: 'IN', value: this.searchParams.type })
      }
      if (!_.isEmpty(this.searchParams.traceId)) {
        params.push({ key: 'traceId', value: _.trim(this.searchParams.traceId) })
      }
      this.$refs.tb.search(params)
    },
    resetSearch() {
      this.searchParams = {
        type: [],
        createTime: getDefaultCreateTimeRange(),
        traceId: ''
      }
      this.search()
    },
    showDetail(record) {
      let content = null
      try {
        content = JSON.stringify(JSON.parse(record.content), null, 2)
      } catch (error) {
        content = record.content
      }
      this.$confirm(<pre class="pre-content">{content}</pre>, {
        title: '详细信息'
      })
    }
  }
}
</script>

<style lang="less" scoped></style>
