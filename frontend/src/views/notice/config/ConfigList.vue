<template>
  <ContentWrap>
    <div class="table-page-search-wrapper">
      <el-form layout="inline">
        <el-row :gutter="48">
          <el-col :md="5" :sm="24">
            <el-form-item label="名称">
              <el-input v-model="searchObj.name" placeholder="请输入" @keyup.enter="search" />
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="类型">
              <el-select v-model="searchObj.type" clearable @change="search">
                <el-option
                  v-for="item in typeList"
                  :key="item.type"
                  :value="item.type"
                  :label="item.name"
                />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="状态">
              <el-select v-model="searchObj.state" clearable @change="search">
                <el-option value="stopped" label="停止" />
                <el-option value="started" label="启动" />
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

    <div class="table-operator">
      <el-button type="primary" v-hasPermi="'notify-config:add'" @click="handleAdd"
        ><Icon icon="el:Plus" />新建</el-button
      >
    </div>

    <PageTable ref="tb" :url="url" :columns="columns" />

    <ConfigAdd ref="modal" @success="handleOk" />
    <NoticeHistory ref="NoticeHistory" />
  </ContentWrap>
</template>

<script lang="jsx">
// import _ from 'lodash-es'
import { configTableUrl, remove, start, stop, copyNotify, configTypes } from '@/views/notice/api.js'
import ConfigAdd from './modules/ConfigAdd.vue'
import NoticeHistory from './modules/NoticeHistory.vue'
import ConfigTableActions from './ConfigTableActions.vue'
export default {
  name: 'NotifyConfigList',
  components: {
    ConfigAdd,
    NoticeHistory
  },
  data() {
    return {
      url: configTableUrl,
      // 查询参数
      searchObj: {
        name: ''
      },
      // 表头
      columns: [
        { label: 'ID', field: 'id' },
        { label: '名称', field: 'name' },
        { label: '通知类型', field: 'type' },
        {
          label: '状态',
          field: 'state',
          slots: {
            default: (data) => {
              if (data.row.state === 'started') {
                return <el-tag type="success">启用</el-tag>
              } else {
                return <el-tag type="info">停用</el-tag>
              }
            }
          }
        },
        { label: '创建时间', field: 'createTime' },
        {
          label: '操作',
          width: '220px',
          field: 'action',
          slots: {
            default: (data) => {
              return (
                <ConfigTableActions
                  record={data.row}
                  onEdit={this.handleEdit}
                  onOk={this.handleOk}
                />
              )
            }
          }
        }
      ],
      typeList: []
    }
  },
  mounted() {
    this.search()
    configTypes().then((result) => {
      this.typeList = result
    })
  },
  methods: {
    search() {
      const condition = []
      if (this.searchObj.name) {
        condition.push({ key: 'name', value: this.searchObj.name, oper: 'LIKE' })
      }
      if (this.searchObj.state) {
        condition.push({ key: 'state', value: this.searchObj.state })
      }
      if (this.searchObj.type) {
        condition.push({ key: 'type', value: this.searchObj.type })
      }
      this.$refs.tb.search(condition)
    },
    resetSearch() {
      this.searchObj = {}
      this.search()
    },
    handleAdd() {
      this.$refs.modal.add()
    },
    handleEdit(record) {
      this.$refs.modal.edit(record)
    },
    handleOk() {
      // 新增/修改 成功时，重载列表
      this.search()
    },
    showHistory(data) {
      this.$refs.NoticeHistory.open(data.id)
    }
  }
}
</script>
