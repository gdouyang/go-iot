<template>
  <span>
    <el-button link type="primary" @click="handleEdit(record.id)">编辑</el-button>
    <span v-hasPermi="'rule-mgr:add'">
      <el-divider direction="vertical" />
      <el-popconfirm title="确认复制？" @confirm="copy(record)">
        <template #reference>
          <el-button link type="primary">复制</el-button>
        </template>
      </el-popconfirm>
    </span>
    <el-divider direction="vertical" />
    <span v-if="record.state === 'stopped'">
      <span v-hasPermi="'rule-mgr:save'">
        <el-popconfirm title="确认启动？" @confirm="start(record)">
          <template #reference>
            <el-button link type="primary">启动</el-button>
          </template>
        </el-popconfirm>
      </span>
      <span v-hasPermi="'rule-mgr:delete'">
        <el-divider direction="vertical" />
        <el-popconfirm title="确认删除？" @confirm="deleteScene(record.id)">
          <template #reference>
            <el-button link type="primary">删除</el-button>
          </template>
        </el-popconfirm>
      </span>
    </span>
    <span v-else v-hasPermi="'rule-mgr:save'">
      <el-popconfirm title="确认停止？" @confirm="stop(record)">
        <template #reference>
          <el-button link type="primary">停止</el-button>
        </template>
      </el-popconfirm>
    </span>
  </span>
</template>

<script lang="jsx">
import { tableUrl, get, remove, start, stop, copy } from './api.js'

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
    start(item) {
      this.spinning = true
      start(item.id).then((resp) => {
        if (resp.success) {
          this.$message.success('启动成功')
          this.handleOk()
        } else {
          this.spinning = false
        }
      })
    },
    deleteScene(id) {
      this.spinning = true
      remove(id).then((response) => {
        if (response.success) {
          this.$message.success('操作成功')
          this.handleOk()
        } else {
          this.spinning = false
        }
      })
    },
    stop(item) {
      this.spinning = true
      stop(item.id).then((response) => {
        if (response.success) {
          this.$message.success('停止成功')
          this.handleOk()
        } else {
          this.spinning = false
        }
      })
    },
    copy(item) {
      copy(item.id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.handleOk()
        }
      })
    }
  }
}
</script>
