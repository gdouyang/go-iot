<template>
  <span>
    <el-button link type="primary" @click="handleEdit(record)">编辑</el-button>
    <span>
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
import { removeRole } from '../api.js'

export default {
  name: 'RoleActions',
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
    remove(row) {
      removeRole(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.$emit('ok')
        }
      })
    }
  }
}
</script>
