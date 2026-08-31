<template>
  <span>
    <el-button link type="primary" @click="detail(record.id)" v-hasPermi="'product-mgr:add'"
      >查看</el-button
    >
    <span v-hasPermi="'product-mgr:add'">
      <el-divider direction="vertical" />
      <el-link type="primary" :href="getExportUrl(record.id)" target="_blank">导出</el-link>
    </span>
    <span v-hasPermi="'product-mgr:save'">
      <el-divider direction="vertical" />
      <el-popconfirm v-if="record.state" title="确认停用？" @confirm="unDeploy(record.id)">
        <template #reference>
          <el-button link type="primary">停用</el-button>
        </template>
      </el-popconfirm>
      <el-button link type="primary" @click="deploy(record.id)" v-else>发布</el-button>
    </span>
    <span v-if="!record.state" v-hasPermi="'product-mgr:delete'">
      <el-divider direction="vertical" />
      <el-popconfirm title="确认删除？" @confirm="deleteById(record.id)">
        <template #reference>
          <el-button link type="primary">删除</el-button>
        </template>
      </el-popconfirm>
    </span>
  </span>
</template>

<script lang="jsx">
import { deploy, undeploy, remove, getExportUrl } from './api.js'

export default {
  name: 'TableActions',
  components: {},
  props: {
    record: {
      type: Object,
      default: () => ({})
    }
  },
  data() {
    return {}
  },
  mounted() {},
  methods: {
    handleEdit(record) {
      this.$emit('edit', record)
    },
    handleOk() {
      this.$emit('ok')
    },
    detail(id) {
      this.$emit('edit', id)
    },
    deploy(id) {
      deploy(id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    },
    unDeploy(id) {
      undeploy(id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    },
    deleteById(id) {
      remove(id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    },
    getExportUrl(id) {
      return getExportUrl(id)
    }
  }
}
</script>
