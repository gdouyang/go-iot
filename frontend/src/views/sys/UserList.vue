<template>
  <ContentWrap>
    <div class="table-page-search-wrapper">
      <el-form layout="inline">
        <el-row :gutter="48">
          <el-col :md="5" :sm="24">
            <el-form-item label="账号">
              <el-input v-model="searchObj.username" placeholder="请输入" @keyup.enter="search" />
            </el-form-item>
          </el-col>
          <el-col :md="5" :sm="24">
            <el-form-item label="名称">
              <el-input v-model="searchObj.nickname" placeholder="请输入" @keyup.enter="search" />
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
      <el-button type="primary" @click="handleAdd" v-hasPermi="'user-mgr:add'"
        ><Icon icon="el:Plus" />新建</el-button
      >
    </div>

    <PageTable ref="tb" :url="url" :columns="columns" />

    <user-modal ref="modal" @ok="handleOk" :showTanent="showTanent" />
  </ContentWrap>
</template>

<script lang="jsx">
import { userTableUrl, enableUser, disableUser, removeUser } from './api.js'
import UserModal from './modules/UserModal.vue'
import UserActions from './modules/UserActions.vue'

export default {
  name: 'UserList',
  components: {
    UserModal
  },
  data() {
    return {
      url: userTableUrl,
      showTanent: false,
      // 查询参数
      searchObj: {},
      // 表头
      columns: [
        { label: 'ID', field: 'id', width: '150px' },
        { label: '账号', field: 'username' },
        { label: '名称', field: 'nickname' },
        {
          label: '状态',
          field: 'enableFlag',
          slots: {
            default: (data) => {
              if (data.row.enableFlag) {
                return <el-tag type="success">启用</el-tag>
              } else {
                return <el-tag type="info">禁用</el-tag>
              }
            }
          }
        },
        { label: '描述', field: 'desc' },
        { label: '创建时间', field: 'createTime' },
        {
          label: '操作',
          width: 210,
          field: 'action',
          slots: {
            default: (data) => {
              return <UserActions record={data.row} onEdit={this.handleEdit} onOk={this.handleOk} />
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
      if (this.searchObj.username) {
        condition.push({ key: 'username', value: this.searchObj.username, oper: 'LIKE' })
      }
      if (this.searchObj.nickname) {
        condition.push({ key: 'nickname', value: this.searchObj.nickname, oper: 'LIKE' })
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
    remove(row) {
      removeUser(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    }
  }
}
</script>
