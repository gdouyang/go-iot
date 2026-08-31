<template>
  <span>
    <el-button link type="primary" @click="meters(record)">查看</el-button>
    <span v-hasPermi="'network-config:save'">
      <el-divider direction="vertical" />
      <el-button link type="primary" @click="handleEdit(record)">编辑</el-button>
    </span>
    <span v-if="!record.productId" v-hasPermi="'network-config:delete'">
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
import { removeNetwork, meters } from '../networkapi.js'

export default {
  name: 'NetworkActions',
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
    meters(row) {
      meters(row.id)
    },
    remove(row) {
      removeNetwork(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.$emit('ok')
        }
      })
    }
  }
}
</script>
