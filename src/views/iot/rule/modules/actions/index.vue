<template>
  <el-card size="small" shadow="never" class="mt-5px">
    <el-row>
      <span style="margin-right: 10px">执行动作{{ position + 1 }}</span>
      <el-popconfirm
        :title="`确认删除此执行动作${position + 1}？`"
        @confirm="$emit('remove', position)"
      >
        <template #reference>
          <el-button link type="primary">删除</el-button>
        </template>
      </el-popconfirm>
    </el-row>

    <el-row :gutter="16" :key="position + 1">
      <el-col :span="6">
        <el-select
          placeholder="选择动作类型"
          v-model="action.executor"
          key="trigger"
          @change="executorChange"
          :class="{ 'v-error': hasError }"
        >
          <el-option value="notifier" label="消息通知" />
          <el-option value="device-message-sender" label="设备输出" />
        </el-select>
      </el-col>
      <template v-if="action.executor === 'notifier'">
        <NotifierAction :actionData="action" />
      </template>
      <template v-else-if="action.executor === 'device-message-sender'">
        <DeviceAction :actionData="action" />
      </template>
    </el-row>
  </el-card>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { newNotifierAction, newDeviceMessageAction } from './data.js'
import NotifierAction from './NotifierAction.vue'
import DeviceAction from './DeviceAction.vue'
export default {
  name: 'Actions',
  components: {
    NotifierAction,
    DeviceAction
  },
  inject: ['formChecker'],
  props: {
    action: {
      type: Object,
      default: null
    },
    position: {
      type: Number,
      default: null
    }
  },
  data() {
    return {
      hasError: false
    }
  },
  created() {
    const checkerId = (this.checkerId = 'action' + _.uniqueId())
    this.formChecker.set(checkerId, () => {
      this.hasError = false
      if (!this.action.executor) {
        this.hasError = true
        return false
      }
      return true
    })
  },
  beforeUnmount() {
    this.formChecker.delete(this.checkerId)
  },
  methods: {
    executorChange(value) {
      if (value === 'notifier') {
        this.action.configuration = newNotifierAction().configuration
      } else if (value === 'device-message-sender') {
        this.action.configuration = newDeviceMessageAction().configuration
      }
      // this.action.executor = value
      this.$emit('save', this.position, this.action)
    }
  }
}
</script>
