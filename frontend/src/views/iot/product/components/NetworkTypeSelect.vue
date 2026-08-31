<template>
  <el-select
    :placeholder="placeholder"
    v-model="internalValue"
    :disabled="disabled"
    v-bind="$attrs"
  >
    <el-option value="MQTT_BROKER" label="自定义MQTT_BROKER" />
    <el-option value="GOIOT_MQTT_BROKER" label="GOIOT_MQTT_BROKER" />
    <el-option value="TCP_SERVER" label="自定义TCP_SERVER" />
    <el-option value="HTTP_SERVER" label="自定义HTTP_SERVER" />
    <el-option value="COAP_SERVER" label="自定义COAP_SERVER" />
    <el-option value="WEBSOCKET_SERVER" label="自定义WEBSOCKET_SERVER" />
    <el-option value="MQTT_CLIENT" label="自定义MQTT_CLIENT" />
    <el-option value="TCP_CLIENT" label="自定义TCP_CLIENT" />
    <el-option value="MODBUS" label="自定义MODBUS_TCP" />
  </el-select>
</template>

<script>
export default {
  name: 'NetworkTypeSelect',
  inheritAttrs: false, // 添加这行以防止属性被添加到根元素
  props: {
    // 查询场景常为 undefined，允许空串/空值
    modelValue: {
      type: String,
      default: ''
    },
    disabled: {
      type: Boolean,
      default: false
    },
    placeholder: {
      type: String,
      default: '请选择'
    }
  },
  emits: ['update:modelValue'],
  computed: {
    internalValue: {
      get() {
        return this.modelValue ?? ''
      },
      set(value) {
        this.$emit('update:modelValue', value ?? '')
      }
    }
  }
}
</script>
