<template>
  <div>
    <ContentWrap>
      <div>
        <el-form label-width="auto">
          <el-row>
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
      <PageTable ref="tb" :columns="columns" :url="tableUrl" emptyText="暂无属性数据" />
    </ContentWrap>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import dayjs from 'dayjs'
import { getDevicePropertysUrl } from '@/views/iot/device/api.js'
import { getDefaultCreateTimeRange, defaultTime, dateShortcuts } from './dateRange.js'

export default {
  name: 'Properties',
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
      }
    }
  },
  created() {
    this.tableUrl = getDevicePropertysUrl(this.device.id)
  },
  mounted() {
    let properties = []
    try {
      const metadata = this.device?.metadata
      if (metadata) {
        const parsed = typeof metadata === 'string' ? JSON.parse(metadata) : metadata
        properties = parsed?.properties || []
      }
    } catch (e) {
      console.error('解析设备物模型失败', e)
    }
    const columns = []
    _.forEach(properties, (prop) => {
      columns.push({ field: prop.id, label: prop.name })
    })
    columns.push({ field: 'createTime', label: '时间', minWidth: '120px' })
    this.columns = columns
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
        // 后端要求格式：YYYY-MM-DD HH:mm:ss，oper=BTW，value 用逗号拼接起止时间
        const formatDate = this.searchParams.createTime.map((e) =>
          dayjs(e).format('YYYY-MM-DD HH:mm:ss')
        )
        params.push({ key: 'createTime', oper: 'BTW', value: formatDate.join(',') })
      }
      this.$refs.tb.search(params)
    },
    resetSearch() {
      // 重置回与后端索引默认范围一致的「最近 1 个月」
      this.searchParams = {
        createTime: getDefaultCreateTimeRange()
      }
      this.search()
    }
  }
}
</script>

<style lang="less" scoped></style>
