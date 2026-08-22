<template>
  <div>
    <div class="flex justify-end mb-3">
      <el-button type="primary" @click="add">
        <Icon icon="el:Plus" class="mr-1" />
        添加事件
      </el-button>
    </div>
    <el-table rowKey="id" :data="data" border style="width: 100%" max-height="calc(100vh - 320px)">
      <el-table-column prop="id" label="事件标识" min-width="120" />
      <el-table-column prop="name" label="事件名称" min-width="140" />
      <el-table-column prop="type" label="数据类型" width="120" />
      <el-table-column prop="description" label="说明" min-width="160" show-overflow-tooltip />
      <el-table-column label="操作" width="130" fixed="right">
        <template #default="scope">
          <el-button link type="primary" @click="edit(scope.row)">修改</el-button>
          <el-divider direction="vertical" />
          <el-popconfirm title="确认删除？" @confirm="remove(scope.row)">
            <template #reference>
              <el-button link type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
    <EventsAdd v-if="visible" :data="current" @save="saveData" @close="close" />
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import EventsAdd from './add/EventsAdd.vue'
export default {
  name: 'Events',
  components: {
    EventsAdd
  },
  props: {
    data: {
      type: Array,
      default: () => []
    }
  },
  data() {
    return {
      gradeText: {
        ordinary: '普通',
        warn: '警告',
        urgent: '紧急'
      },
      visible: false,
      current: {},
      isEdit: false
    }
  },
  mounted() {},
  methods: {
    add() {
      this.isEdit = false
      this.current = {}
      this.visible = true
    },
    edit(item) {
      this.isEdit = true
      this.current = _.cloneDeep(item)
      this.visible = true
    },
    remove(item) {
      const temp = this.data.filter((e) => e.id !== item.id)
      this.$emit('save', temp)
    },
    saveData(item, onlySave) {
      const data = this.data || []
      const i = data.findIndex(
        (j) => String(j.id || '').toLowerCase() === String(item.id || '').toLowerCase()
      )
      if (i > -1) {
        if (!this.isEdit) {
          this.$message.error('事件标识已存在（不区分大小写），请修改')
          return
        }
        data[i] = item
      } else {
        data.push(item)
      }
      this.$emit('save', data, onlySave)
      this.close()
    },
    close() {
      this.current = {}
      this.visible = false
    }
  }
}
</script>

<style lang="less" scoped></style>
