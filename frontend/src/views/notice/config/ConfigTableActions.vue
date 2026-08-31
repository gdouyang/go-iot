<template>
  <span>
    <el-button link type="primary" @click="handleEdit(record)" v-hasPermi="'notify-config:save'"
      >编辑</el-button
    >
    <span v-hasPermi="'notify-config:delete'">
      <el-divider direction="vertical" />
      <el-popconfirm title="确认删除？" @confirm="remove(record)">
        <template #reference>
          <el-button link type="primary">删除</el-button>
        </template>
      </el-popconfirm>
    </span>
    <span v-hasPermi="'notify-config:add'">
      <el-divider direction="vertical" />
      <el-popconfirm title="确认复制？" @confirm="copy(record)">
        <template #reference>
          <el-button link type="primary">复制</el-button>
        </template>
      </el-popconfirm>
    </span>
    <span v-hasPermi="'notify-config:save'">
      <el-divider direction="vertical" />
      <el-popconfirm title="确认停止？" @confirm="stop(record)" v-if="record.state !== 'stopped'">
        <template #reference>
          <el-button link type="primary">停止</el-button>
        </template>
      </el-popconfirm>
      <el-popconfirm title="确认启动？" @confirm="start(record)" v-else>
        <template #reference>
          <el-button link type="primary">启动</el-button>
        </template>
      </el-popconfirm>
    </span>
  </span>
</template>

<script lang="jsx">
import { remove, start, stop } from '@/views/notice/api.js'

export default {
  name: 'ConfitTableActions',
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
    remove(row) {
      remove(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    },
    copy(row) {
      copyNotify(row.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    },
    start(item) {
      start(item.id).then((resp) => {
        if (resp.success) {
          this.$message.success('启动成功')
          this.handleOk()
        }
      })
    },
    stop(item) {
      stop(item.id).then((response) => {
        if (response.success) {
          this.$message.success('停止成功')
          this.handleOk()
        }
      })
    }
  }
}
</script>
