<template>
  <ContentWrap>
    <div class="table-page-search-wrapper">
      <el-form layout="inline">
        <el-row :gutter="48">
          <el-col :md="5" :sm="24">
            <el-form-item label="名称">
              <el-input v-model="searchObj.name" placeholder="请输入" />
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="端口">
              <el-input v-model="searchObj.port" placeholder="请输入" />
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="状态">
              <el-select v-model="searchObj.state" clearable>
                <el-option value="runing" label="运行中" />
                <el-option value="stop" label="停用" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="网络类型">
              <network-type-select v-model="searchObj.type" clearable :disabled="false" />
            </el-form-item>
          </el-col>
          <el-col :md="4" :sm="24">
            <span class="table-page-search-submitButtons">
              <el-button type="primary" @click="search">查询</el-button>
              <el-button style="margin-left: 8px" @click="resetSearch">重置</el-button>
            </span>
          </el-col>
        </el-row>
      </el-form>
    </div>
    <div class="table-operator">
      <el-button type="primary" @click="handleAdd" v-hasPermi="'network-config:add'"
        ><Icon icon="el:Plus" />新建</el-button
      >
    </div>
    <PageTable ref="tb" :url="tableUrl" :columns="columns" />
    <NetworkModal ref="modal" @ok="handleOk" />
  </ContentWrap>
</template>

<script lang="jsx">
import { tableUrl, removeNetwork } from './networkapi.js'
import NetworkModal from './modules/NetworkModal.vue'
import NetworkActions from './modules/NetworkActions.vue'
import NetworkTypeSelect from '../iot/product/components/NetworkTypeSelect.vue'

const defaultSearchObj = {
  name: '',
  port: '',
  state: '',
  type: ''
}

export default {
  name: 'NetworkList',
  components: {
    NetworkModal,
    NetworkTypeSelect
  },
  data() {
    return {
      tableUrl: tableUrl,
      // 查询参数（type 默认空串，避免 NetworkTypeSelect 收到 undefined）
      searchObj: { ...defaultSearchObj },
      // 表头
      columns: [
        { label: 'ID', field: 'id', width: '150px' },
        { label: '关联产品', field: 'productId' },
        { label: '网络类型', field: 'type' },
        { label: '端口', field: 'port' },
        {
          label: '状态',
          field: 'state',
          slots: {
            default: (data) => {
              if (data.row.state === 'runing') {
                return <el-tag type="success">运行中</el-tag>
              } else {
                return <el-tag type="info">停止</el-tag>
              }
            }
          }
        },
        { label: '创建时间', field: 'createTime' },
        {
          label: '操作',
          width: '180px',
          field: 'action',
          slots: {
            default: (data) => {
              return (
                <NetworkActions record={data.row} onEdit={this.handleEdit} onOk={this.handleOk} />
              )
            }
          }
        }
      ]
    }
  },
  computed: {},
  mounted() {
    this.search()
  },
  methods: {
    search() {
      const condition = []
      if (this.searchObj.port) {
        condition.push({ key: 'port', value: this.searchObj.port })
      }
      if (this.searchObj.state) {
        condition.push({ key: 'state', value: this.searchObj.state })
      }
      if (this.searchObj.type) {
        condition.push({ key: 'type', value: this.searchObj.type })
      }
      if (this.searchObj.name) {
        condition.push({ key: 'name', value: this.searchObj.name, oper: 'LIKE' })
      }
      this.$refs.tb.search(condition)
    },
    resetSearch() {
      this.searchObj = { ...defaultSearchObj }
      this.search()
    },
    handleAdd() {
      this.$refs.modal.add()
    },
    handleEdit(record) {
      this.$refs.modal.edit(record)
    },
    handleOk() {
      this.resetSearch()
    }
  }
}
</script>
