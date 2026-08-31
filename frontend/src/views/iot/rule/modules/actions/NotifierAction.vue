<template>
  <el-col :span="4">
    <el-tooltip content="请选择通知类型">
      <el-select
        placeholder="选择通知类型"
        v-model="actionData.configuration.notifyType"
        :class="{ 'v-error': notifyTypeError }"
        @change="notifyTypeChange"
      >
        <el-option
          v-for="item in notifyTypeConfig"
          :key="item.type"
          :value="item.type"
          :label="item.name"
        />
      </el-select>
    </el-tooltip>
  </el-col>
  <el-col :span="4">
    <el-tooltip content="请选择通知配置">
      <el-select
        placeholder="选择通知配置"
        v-model="actionData.configuration.notifierId"
        :class="{ 'v-error': notifierIdError }"
      >
        <el-option
          v-for="item in messageConfig"
          :key="item.id"
          :value="item.id"
          :label="item.name"
        />
      </el-select>
    </el-tooltip>
  </el-col>
</template>

<script lang="jsx">
import _ from 'lodash-es'
// import encodeQueryParam from '@/utils/encodeParam.js'
import { configTypes, listAll } from '@/views/notice/api.js'
export default {
  name: 'NotifierAction',
  components: {},
  inject: ['formChecker'],
  props: {
    actionData: {
      type: Object,
      default: null
    }
  },
  data() {
    return {
      notifyTypeConfig: [],
      messageConfig: [],
      templateConfig: [],
      notifyTypeError: false,
      notifierIdError: false,
      templateIdError: false
    }
  },
  created() {
    this.getAllTypes()
    const checkerId = (this.checkerId = 'notifier-action' + _.uniqueId())
    this.formChecker.set(checkerId, () => {
      this.notifyTypeError = this.notifierIdError = false
      if (!this.actionData.configuration.notifyType) {
        this.notifyTypeError = true
      }
      if (!this.actionData.configuration.notifierId) {
        this.notifierIdError = true
      }
      if (this.notifyTypeError || this.notifierIdError) {
        return false
      }
      return true
    })
  },
  beforeUnmount() {
    this.formChecker.delete(this.checkerId)
  },
  methods: {
    getAllTypes() {
      const actionData = this.actionData
      configTypes().then((result) => {
        this.notifyTypeConfig = result
        if (actionData.configuration.notifyType) {
          this.findNotifier(actionData.configuration.notifyType)
        }
      })
    },
    notifyTypeChange(value) {
      this.findNotifier(value)
      this.actionData.configuration.notifyType = value
      this.clearNotifierId()
    },
    clearNotifierId() {
      this.messageConfig = []
      this.actionData.configuration.notifierId = undefined
    },
    findNotifier(value) {
      // 通知配置
      const param = {
        type: value,
        state: 'started'
      }
      listAll(param).then((response) => {
        if (response.success) {
          this.messageConfig = response.result
        }
      })
    }
  }
}
</script>
<style lang="less" scoped></style>
