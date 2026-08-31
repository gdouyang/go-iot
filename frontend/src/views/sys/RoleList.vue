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
            <span class="table-page-search-submitButtons">
              <el-button type="primary" @click="search">查询</el-button>
              <el-button style="margin-left: 8px" @click="resetSearch">重置</el-button>
            </span>
          </el-col>
        </el-row>
      </el-form>
    </div>

    <div class="table-operator">
      <el-button type="primary" @click="handleAdd" v-hasPermi="'role-mgr:add'"
        ><Icon icon="el:Plus" />新建</el-button
      >
    </div>

    <PageTable ref="tb" :url="url" :columns="columns" />

    <role-modal ref="modal" @ok="handleOk" />
  </ContentWrap>
</template>

<script lang="jsx">
import { roleTableUrl, removeRole } from './api.js'
import RoleModal from './modules/RoleModal.vue'
import RoleActions from './modules/RoleActions.vue'

export default {
  name: 'RoleList',
  components: {
    RoleModal
  },
  data() {
    return {
      url: roleTableUrl,
      // 查询参数
      searchObj: {},
      // 表头
      columns: [
        { label: 'ID', field: 'id', width: '150px' },
        { label: '角色名称', field: 'name' },
        { label: '描述', field: 'desc' },
        { label: '创建时间', field: 'createTime' },
        {
          label: '操作',
          minWidth: '110px',
          field: 'action',
          slots: {
            default: (data) => {
              return <RoleActions record={data.row} onEdit={this.handleEdit} onOk={this.handleOk} />
            }
          }
        }
      ]
    }
  },
  mounted() {
    this.search()
  },
  methods: {
    search() {
      const condition = []
      if (this.searchObj.name) {
        condition.push({ key: 'name', value: this.searchObj.name, oper: 'LIKE' })
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
    }
  }
}
</script>
