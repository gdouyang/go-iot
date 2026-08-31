<template>
  <div v-show="!openModal">
    <ContentWrap>
      <div class="table-page-search-wrapper">
        <el-form layout="inline">
          <el-row :gutter="48">
            <el-col :md="5" :sm="24">
              <el-form-item label="名称">
                <el-input
                  v-model="searchObj.name"
                  clearable
                  placeholder="请输入"
                  @keyup.enter="search"
                />
              </el-form-item>
            </el-col>
            <el-col :md="5" :sm="24">
              <el-form-item label="状态">
                <el-select
                  v-model="searchObj.state"
                  clearable
                  placeholder="请选择"
                  @change="search"
                >
                  <el-option value="stopped" label="停止" />
                  <el-option value="started" label="启动" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :md="8" :sm="24">
              <span class="table-page-search-submitButtons">
                <el-button type="primary" @click="search">查询</el-button>
                <el-button style="margin-left: 8px" @click="resetSearch">重置</el-button>
              </span>
            </el-col>
          </el-row>
        </el-form>
      </div>

      <div class="table-operator">
        <el-button type="primary" @click="handleAdd" v-hasPermi="'rule-mgr:add'"
          ><Icon icon="el:Plus" />新建</el-button
        >
      </div>

      <PageTable ref="tb" :url="url" :columns="columns" />
    </ContentWrap>
  </div>
  <div v-if="openModal">
    <RuleAdd @success="back" @close="back" />
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { tableUrl, get, remove, start, stop, copy } from './api.js'
import TableActions from './TableActions.vue'
import RuleAdd from './modules/RuleAdd.vue'

export default {
  name: 'SceneList',
  components: {
    RuleAdd
  },
  data() {
    return {
      url: tableUrl,
      // 查询参数
      searchObj: {},
      // 表头
      columns: [
        { label: 'ID', field: 'id', width: 150 },
        { label: '名称', field: 'name' },
        {
          label: '状态',
          field: 'state',
          slots: {
            default: (data) => {
              if (data.row.state == 'started') {
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
          width: 230,
          field: 'action',
          slots: {
            default: (data) => {
              return (
                <TableActions record={data.row} onEdit={this.handleEdit} onOk={this.handleOk} />
              )
            }
          }
        }
      ],
      openModal: false,
      isEdit: false
    }
  },
  mounted() {
    const id = this.$route.query.id
    if (id) {
      this.handleEdit(id)
    } else {
      this.search()
    }
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
      this.$refs.tb.search(condition)
    },
    resetSearch() {
      this.searchObj = {}
      this.search()
    },
    tableRefresh() {
      this.$refs.tb.refresh()
    },
    handleAdd() {
      this.$router.push({ name: this.$route.name, query: { id: 'add' } }).then(() => {
        this.openModal = true
        this.isEdit = false
      })
    },
    handleEdit(id) {
      this.$router.push({ name: this.$route.name, query: { id: id } }).then(() => {
        this.openModal = true
        this.isEdit = true
      })
    },
    handleOk() {
      this.openModal = false
      // 新增/修改 成功时，重载列表
      this.search()
    },
    back() {
      this.$router.push({ name: this.$route.name, query: {} }).then(() => {
        if (this.isEdit) {
          this.openModal = false
          this.tableRefresh()
        } else {
          this.handleOk()
        }
      })
    }
  }
}
</script>
