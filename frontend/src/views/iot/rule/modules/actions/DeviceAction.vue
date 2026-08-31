<template>
  <el-col :span="4">
    <el-input placeholder="点击选择设备" v-model="deviceData.name" readonly>
      <template #append>
        <BaseButton @click="selectDevice"><Icon icon="el:Link" title="点击选择设备" /> </BaseButton>
      </template>
    </el-input>
  </el-col>
  <el-col v-if="deviceData.name" :span="4">
    <el-select
      placeholder="选择类型，如：属性/功能"
      :model-value="messageTypeDefaultValue"
      @change="messageTypeChange"
    >
      <!-- <el-option value="WRITE_PROPERTY">设置属性</el-option> -->
      <el-option value="INVOKE_FUNCTION" label="调用功能" />
    </el-select>
  </el-col>
  <!-- <div v-show="messageType === 'WRITE_PROPERTY'">
      <el-col :span="4">
        <el-select
          placeholder="物模型属性"
          :value="propertiesDefaultValue"
          @change="propertiesDataChange"
        >
          <el-option v-for="item in properties" :key="item.id" :value="item.id">{{ item.name }}({{ item.id }})</el-option>
        </el-select>
      </el-col>
      <Properties :propertiesData="propertiesData" :actionData="actionData" :arrayData.sync="arrayData" />
    </div> -->
  <template v-if="messageType === 'INVOKE_FUNCTION'">
    <el-col :span="4">
      <el-select
        placeholder="物模型功能"
        :model-value="functionDefaultValue"
        @change="functionIdChange"
      >
        <el-option
          v-for="item in functions"
          :key="item.id"
          :value="item.id"
          :label="`${item.name}(${item.id})`"
        />
      </el-select>
    </el-col>
    <el-col :span="24">
      <el-row
        v-for="(item, index) in functionData.inputs"
        :key="`function_${item.id}_${index}`"
        :gutter="16"
        style="margin-top: 5px"
      >
        <el-col :span="4">
          <el-input :model-value="`${item.name}(${item.id})`" readonly />
        </el-col>
        <DeviceFunction :item="item" :index="index" :actionData="actionData" />
      </el-row>
    </el-col>
  </template>

  <DeviceSelect @select="select" ref="DeviceSelect" />
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { WRITE_PROPERTY, INVOKE_FUNCTION } from './data.js'
import Properties from './DeviceProperties.vue'
import DeviceFunction from './DeviceFunction.vue'
import DeviceSelect from '@/views/iot/device/DeviceSelect.vue'
import { get as getDevice } from '@/views/iot/device/api.js'
export default {
  name: 'DeviceAction',
  components: {
    Properties,
    DeviceFunction,
    DeviceSelect
  },
  props: {
    actionData: {
      type: Object,
      default: null
    }
  },
  data() {
    return {
      messageType: null,
      deviceData: {
        name: null,
        metadata: {
          properties: [],
          functions: []
        }
      },
      propertiesData: {},
      arrayData: [],
      functionData: {},
      properties: [],
      functions: []
    }
  },
  computed: {
    messageTypeDefaultValue() {
      const message = this.actionData.configuration
      return (message && message.messageType) || undefined
    },
    propertiesDefaultValue() {
      return this.propertiesData.id || undefined
    },
    functionDefaultValue() {
      const message = this.actionData.configuration
      return (message && message.functionId) || undefined
    }
  },
  mounted() {
    const deviceId = this.actionData.configuration.deviceId
    if (deviceId) {
      this.findDevice(deviceId)
    }
  },
  methods: {
    messageTypeChange(value) {
      this.messageType = value
      if (value === 'WRITE_PROPERTY') {
        this.actionData.configuration = WRITE_PROPERTY()
      } else if (value === 'INVOKE_FUNCTION') {
        this.actionData.configuration = INVOKE_FUNCTION()
      }
      this.propertiesData = {}
      this.functionData = {}
      this.actionData.configuration.messageType = value
    },
    selectDevice() {
      this.$refs.DeviceSelect.open()
    },
    select(item) {
      if (item && item.id) {
        this.findDevice(item.id)
      }
    },
    findDevice(deviceId) {
      return getDevice(deviceId).then((data) => {
        if (data.success) {
          const result = data.result
          if (result.metadata) {
            result.metadata = JSON.parse(result.metadata)
          } else {
            result.metadata = {
              properties: [],
              functions: []
            }
          }
          this.deviceData = result
          this.properties = result.metadata.properties
          this.functions = result.metadata.functions
          const actionData = this.actionData
          if (!actionData.configuration) {
            actionData.configuration = {}
          }
          this.arrayData = [undefined]
          const message = actionData.configuration
          this.messageType = message.messageType
          if (actionData.configuration.deviceId) {
            if (message.messageType === 'WRITE_PROPERTY') {
              _.forEach(result.metadata.properties, (item) => {
                if (item.id === Object.keys(message.properties)[0]) {
                  this.propertiesData = item
                  if (item.type === 'array') {
                    this.arrayData = message.properties[item.id]
                  }
                }
              })
            } else {
              _.forEach(result.metadata.functions, (item) => {
                if (item.id === message.functionId) {
                  this.functionData = item
                }
              })
            }
          }
          actionData.configuration.deviceId = deviceId
          actionData.configuration.productId = result.productId
        }
      })
    },
    propertiesDataChange(value) {
      const find = _.find(this.properties, (p) => p.id === value)
      if (find) {
        this.propertiesData = find
      }
      this.arrayData = [undefined]
      // 清空服务调用
      this.actionData.configuration.functionId = null
      this.actionData.configuration.data = {}
    },
    functionIdChange(value) {
      const find = _.find(this.functions, (p) => p.id === value)
      this.actionData.configuration.data = {}
      if (find) {
        this.functionData = find
        _.forEach(find.inputs, (func) => {
          this.actionData.configuration.data[func.id] = null
        })
      }
      this.actionData.configuration.functionId = value
    }
  }
}
</script>
