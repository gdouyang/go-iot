<template>
  <span>
    <el-button link type="primary" @click="detail(record.id)">查看</el-button>
    <span v-hasPermi="'device-mgr:save'">
      <el-divider direction="vertical" />
      <el-button link type="primary" @click="handleEdit(record)">修改</el-button>
    </span>
    <span v-hasPermi="'device-mgr:save'">
      <el-divider direction="vertical" />
      <el-button link type="primary" @click="deploy(record.id)" v-if="record.state === 'noActive'"
        >激活</el-button
      >
      <el-popconfirm v-else title="确认停用？" @confirm="unDeploy(record.id)">
        <template #reference>
          <el-button link type="primary">停用</el-button>
        </template>
      </el-popconfirm>
    </span>
    <span v-if="record.state === 'noActive'" v-hasPermi="'device-mgr:delete'">
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
import { remove, undeploy, deploy } from './api.js'

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
      this.$emit('detail', id)
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
    remove(row) {
      remove(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    }
  }
}
</script>
