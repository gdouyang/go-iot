<template>
  <span>
    <el-button link type="primary" @click="handleEdit(record)" v-hasPermi="'user-mgr:save'"
      >编辑</el-button
    >
    <span v-hasPermi="'user-mgr:save'">
      <el-divider direction="vertical" />
      <el-popconfirm
        title="确认启用？"
        v-if="record.enableFlag === false"
        @confirm="enable(record)"
      >
        <template #reference>
          <el-button link type="primary">启用</el-button>
        </template>
      </el-popconfirm>
      <el-popconfirm title="确认禁用？" v-else @confirm="disable(record)">
        <template #reference>
          <el-button link type="primary">禁用</el-button>
        </template>
      </el-popconfirm>
    </span>
    <span v-hasPermi="'user-mgr:delete'">
      <el-divider direction="vertical" />
      <el-popconfirm title="确认删除？" @confirm="remove(record)">
        <template #reference>
          <el-button link type="primary">删除</el-button>
        </template>
      </el-popconfirm>
    </span>
  </span>
</template>

<script lang="jsx">
import { enableUser, disableUser, removeUser } from '../api.js'

export default {
  name: 'Userctions',
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
    enable(row) {
      enableUser(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.$emit('ok')
        }
      })
    },
    disable(row) {
      disableUser(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.$emit('ok')
        }
      })
    },
    remove(row) {
      removeUser(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.$emit('ok')
        }
      })
    }
  }
}
</script>
