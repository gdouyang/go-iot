<template>
  <Dialog ref="addModal" @confirm="addConfirm" @close="addClose" maxHeight="auto">
    <PageTable ref="tb" :url="url" :columns="columns">
      <span slot="notifyTime" slot-scope="text">
        {{ text ? $dayjs(text).format('YYYY-MM-DD HH:mm:ss') : '/' }}
      </span>
      <span slot="state" slot-scope="text, record">
        <el-tag :color="colorMap.get(text.value)">{{ text.text }}</el-tag>
        <el-button link type="primary" v-if="text.value === 'error'" @click="showError(record)"
          >查看</el-button
        >
      </span>
      <span slot="action" slot-scope="text, record">
        <el-button link type="primary" @click="showDetail(record)">查看数据</el-button>
      </span>
    </PageTable>
  </Dialog>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { historyTableUrl } from '@/views/notice/api.js'
const defaultAddObj = {
  id: null,
  name: '',
  configuration: {
    properties: []
  }
}
const colorMap = new Map()
colorMap.set('error', '#f50')
colorMap.set('success', '#108ee9')
export default {
  name: 'NoticeHistory',
  data() {
    return {
      url: historyTableUrl,
      labelCol: {
        xs: { span: 24 },
        sm: { span: 5 }
      },
      wrapperCol: {
        xs: { span: 24 },
        sm: { span: 16 }
      },
      addObj: _.cloneDeep(defaultAddObj),
      colorMap: colorMap,
      columns: [
        { label: 'ID', field: 'id' },
        { label: '时间', field: 'notifyTime', scopedSlots: { customRender: 'notifyTime' } },
        { field: 'state', label: '状态', scopedSlots: { customRender: 'state' } },
        {
          label: '操作',
          width: '150px',
          field: 'action',
          scopedSlots: { customRender: 'action' }
        }
      ]
    }
  },
  created() {},
  methods: {
    open(id) {
      this.$refs.addModal.open({ title: '通知记录' })
      this.$nextTick(() => {
        this.$refs.tb.search([{ key: 'notifierId', value: id }])
      })
    },
    addClose() {},
    addConfirm() {
      this.$refs.addModal.close()
    },
    showError(item) {
      this.$confirm(
        <el-textarea
          value={JSON.stringify(item.errorStack || '{}')}
          row={10}
          style={{ height: '200px' }}
        />,
        {
          title: '错误信息'
        }
      )
    },
    showDetail(item) {
      this.$confirm(
        <el-textarea
          value={JSON.stringify(item.context || '{}')}
          row={4}
          style={{ height: '200px' }}
        />,
        {
          title: '详情'
        }
      )
    }
  }
}
</script>
<style lang="less" scoped></style>
