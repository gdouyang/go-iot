<template>
  <div>
    <ContentWrap>
      <el-tabs v-model="activeTab">
        <el-tab-pane label="升级记录" name="jobs">
          <div>
            <div class="table-page-search-wrapper">
              <el-form layout="inline">
                <el-row :gutter="48">
                  <el-col :md="5" :sm="24">
                    <el-form-item label="产品">
                      <el-select
                        v-model="searchObj.productId"
                        placeholder="产品"
                        clearable
                        @change="search"
                      >
                        <el-option
                          v-for="p in productList"
                          :key="p.id"
                          :value="p.id"
                          :label="p.name"
                        />
                      </el-select>
                    </el-form-item>
                  </el-col>
                  <el-col :md="5" :sm="24">
                    <el-form-item label="设备ID">
                      <el-input
                        v-model="searchObj.deviceId"
                        placeholder="请输入设备ID"
                        @keyup.enter="search"
                      />
                    </el-form-item>
                  </el-col>
                  <el-col :md="5" :sm="24">
                    <el-form-item label="状态">
                      <el-select
                        v-model="searchObj.status"
                        clearable
                        placeholder="状态"
                        @change="search"
                      >
                        <el-option value="pending" label="待处理" />
                        <el-option value="in_progress" label="进行中" />
                        <el-option value="success" label="成功" />
                        <el-option value="fail" label="失败" />
                      </el-select>
                    </el-form-item>
                  </el-col>
                  <el-col :md="6" :sm="24">
                    <span class="table-page-search-submitButtons">
                      <el-button type="primary" @click="search">查询</el-button>
                      <el-button style="margin-left: 8px" @click="resetSearch">重置</el-button>
                    </span>
                  </el-col>
                </el-row>
              </el-form>
            </div>
            <div class="table-operator">
              <el-button type="primary" style="margin-left: 8px" @click="showUpdateDialog"
                >OTA升级</el-button
              >
              <el-button type="danger" style="margin-left: 8px" @click="handleBatchDelete"
                >批量删除</el-button
              >
            </div>
            <PageTable ref="tb" :url="tableUrl" :columns="columns" rowKey="id" />
          </div>
        </el-tab-pane>
        <el-tab-pane label="OTA文件管理" name="media">
          <OtaMediaManager />
        </el-tab-pane>
      </el-tabs>
    </ContentWrap>

    <OtaUpdate ref="otaUpdate" @success="search" />
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { otaPageUrl as pageUrl, getProductList, otaDelete } from './api.js'
import OtaUpdate from './modules/OtaUpdate.vue'
import OtaMediaManager from './modules/OtaMediaManager.vue'
import { ElMessage, ElMessageBox } from 'element-plus'

const defautSearchObj = {}
export default {
  components: {
    OtaUpdate,
    OtaMediaManager
  },
  data() {
    return {
      activeTab: 'jobs',
      tableUrl: pageUrl,
      searchObj: _.cloneDeep(defautSearchObj),
      columns: [
        { type: 'selection', width: 55, align: 'center' },
        { label: 'ID', field: 'id' },
        { label: '设备ID', field: 'deviceId' },
        { label: '产品ID', field: 'productId' },
        { label: '文件名', field: 'fileName' },
        {
          label: '文件大小',
          field: 'fileSize',
          slots: {
            default: (data) => {
              return (data.row.fileSize / 1024).toFixed(2) + ' KB'
            }
          }
        },
        {
          label: '进度',
          field: 'progress',
          slots: {
            default: (data) => {
              if (data.row.totalChunks > 0) {
                const percent = Math.floor((data.row.currentChunk / data.row.totalChunks) * 100)
                return percent + '%'
              }
              return '-'
            }
          }
        },
        {
          label: '状态',
          field: 'status',
          slots: {
            default: (data) => {
              const status = data.row.status
              let type = 'info'
              let text = status
              if (status === 'success') {
                type = 'success'
                text = '成功'
              } else if (status === 'fail') {
                type = 'danger'
                text = '失败'
              } else if (status === 'in_progress') {
                type = 'warning'
                text = '进行中'
              } else if (status === 'pending') {
                type = 'info'
                text = '待处理'
              }
              return <el-tag type={type}>{text}</el-tag>
            }
          }
        },
        { label: '消息', field: 'message' },
        { label: '创建时间', field: 'createTime' },
        { label: '更新时间', field: 'updateTime' },
        {
          label: '操作',
          field: 'action',
          slots: {
            default: (data) => {
              return (
                <el-button link type="danger" onClick={() => this.handleDelete(data.row)}>
                  删除
                </el-button>
              )
            }
          }
        }
      ],
      productList: []
    }
  },
  mounted() {
    this.search()
    getProductList().then((resp) => {
      if (resp.success) {
        this.productList = resp.result
      }
    })
  },
  methods: {
    search() {
      const condition = []
      if (this.searchObj.deviceId) {
        condition.push({ key: 'deviceId', value: this.searchObj.deviceId, oper: 'LIKE' })
      }
      if (this.searchObj.productId) {
        condition.push({ key: 'productId', value: this.searchObj.productId })
      }
      if (this.searchObj.status) {
        condition.push({ key: 'status', value: this.searchObj.status })
      }
      this.$refs.tb.search(condition)
    },
    resetSearch() {
      this.searchObj = _.cloneDeep(defautSearchObj)
      this.search()
    },
    showUpdateDialog() {
      this.$refs.otaUpdate.open()
    },
    handleDelete(row) {
      ElMessageBox.confirm('确认删除该记录吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          await otaDelete([row.id])
          ElMessage.success('删除成功')
          this.search()
        } catch (e) {
          // Error handling
        }
      })
    },
    handleBatchDelete() {
      const ids = this.$refs.tb.getSelectedRows('id')
      if (!ids || ids.length === 0) {
        ElMessage.warning('请选择要删除的记录')
        return
      }
      ElMessageBox.confirm(`确认删除选中的 ${ids.length} 条记录吗？`, '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }).then(async () => {
        try {
          await otaDelete(ids)
          ElMessage.success('批量删除成功')
          this.search()
        } catch (e) {
          // Error handling
        }
      })
    }
  }
}
</script>
