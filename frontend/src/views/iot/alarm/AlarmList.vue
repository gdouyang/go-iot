<template>
  <ContentWrap>
    <div class="table-page-search-wrapper">
      <el-form layout="inline">
        <el-row :gutter="48">
          <el-col :md="5" :sm="24">
            <el-form-item label="产品ID">
              <el-input v-model="searchObj.productId" placeholder="请输入" @keyup.enter="search" />
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="设备ID">
              <el-input v-model="searchObj.deviceId" placeholder="请输入" @keyup.enter="search" />
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="状态">
              <el-select v-model="searchObj.state" clearable @change="search">
                <el-option value="solve" label="已处理" />
                <el-option value="unsolve" label="未处理" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <span class="table-page-search-submitButtons">
              <el-button type="primary" @click="search">查询</el-button>
              <el-button style="margin-left: 8px" @click="resetSearch">重置</el-button>
            </span>
          </el-col>
        </el-row>
      </el-form>
    </div>
    <PageTable ref="tb" rowKey="id" :columns="columns" :url="url" />

    <Dialog title="告警处理" ref="dialog" maxHeight="auto" @confirm="submitData">
      <el-form ref="form" :model="currentLog">
        <el-form-item
          prop="desc"
          label="处理结果"
          :rules="[
            { required: true, message: '请输入处理结果' },
            { max: 2000, message: '处理结果不超过2000个字符' }
          ]"
        >
          <el-textarea
            :rows="8"
            v-model="currentLog.desc"
            maxlength="2000"
            placeholder="请输入处理结果"
          />
        </el-form-item>
      </el-form>
    </Dialog>
  </ContentWrap>
</template>
<script lang="jsx">
import { dateUtil } from '@/utils/dateUtil'
import { pageUrl, solveAlarmLog } from './api.js'
export default {
  name: 'AlamrList',
  components: {},
  mixins: [],
  props: {},
  data() {
    return {
      url: pageUrl,
      searchObj: {
        deviceId: undefined,
        productId: undefined
      },
      columns: [
        { label: '告警名称', field: 'alarmName' },
        { label: '设备ID', field: 'deviceId' },
        { label: '产品ID', field: 'productId' },
        { label: '告警时间', field: 'createTime', scopedSlots: { customRender: 'createTime' } },
        {
          label: '告警时间',
          field: 'createTime',
          slots: {
            default: (data) => {
              return dateUtil(data.row.createTime).format('YYYY-MM-DD HH:mm:ss')
            }
          }
        },
        {
          label: '处理状态',
          field: 'state',
          slots: {
            default: (data) => {
              if (data.row.state === 'solve') {
                return <el-tag type="success">已处理</el-tag>
              } else {
                return <el-tag type="info">未处理</el-tag>
              }
            }
          }
        },
        {
          label: '操作',
          field: 'state',
          slots: {
            default: (data) => {
              return (
                <>
                  <el-button link type="primary" onClick={() => this.detail(data.row)}>
                    详情
                  </el-button>
                  {data.row.state !== 'solve' && (
                    <>
                      <el-divider direction="vertical" />
                      <el-button link type="primary" onClick={() => this.edit(data.row)}>
                        处理
                      </el-button>
                    </>
                  )}
                </>
              )
            }
          }
        }
      ],
      alarmLogId: null,
      currentLog: {
        id: null,
        desc: null
      }
    }
  },
  created() {
    this.$nextTick(() => {
      this.search()
    })
  },
  methods: {
    search() {
      const condition = []
      if (this.searchObj.deviceId) {
        condition.push({ key: 'deviceId', value: this.searchObj.deviceId, oper: 'LIKE' })
      }
      if (this.searchObj.state) {
        condition.push({ key: 'state', value: this.searchObj.state })
      }
      this.$refs.tb.search(condition)
    },
    resetSearch() {
      this.searchObj.deviceId = undefined
      this.searchObj.productId = undefined
      this.search()
    },
    detail(record) {
      let content = null
      try {
        content = JSON.stringify(JSON.parse(record.alarmData), null, 2)
      } catch (error) {
        content = record.alarmData
      }
      this.$confirm(
        <div>
          <pre style="padding: 5px;">{content}</pre>
          {record.state === 'solve' && (
            <div>
              <br />
              <br />
              <span style="font-size: 16px;">处理结果：</span>
              <br />
              <p style="padding: 5px;">{record.desc}</p>
            </div>
          )}
        </div>,
        {
          title: '告警数据'
        }
      )
    },
    edit(record) {
      this.currentLog.id = record.id
      this.currentLog.desc = null
      this.$refs.dialog.open()
    },
    submitData() {
      this.$refs.form.validate((valid) => {
        if (valid) {
          const id = this.currentLog.id
          solveAlarmLog(id, this.currentLog).then((response) => {
            if (response.success) {
              this.$message.success('保存成功')
              this.search()
              this.$refs.dialog.close()
            }
          })
        }
      })
    }
  }
}
</script>
