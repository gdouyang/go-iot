<template>
  <el-col :span="4" v-if="properties.type === 'enum'">
    <el-select placeholder="选择属性值" :model-value="defaultValue" @change="selectChange">
      <el-option
        v-if="item in properties.elements"
        :key="item.value"
        :label="`${item.text}（${item.value}）`"
      />
    </el-select>
  </el-col>
  <el-col v-else-if="properties.type === 'bool'" :span="4">
    <el-select placeholder="选择属性值" :model-value="defaultValue" @change="selectChange">
      <el-option
        :key="properties.trueValue"
        :label="`${properties.trueText}（${properties.trueValue}）`"
      />
      <el-option
        :key="properties.falseValue"
        :label="`${properties.falseText}（${properties.falseValue}）`"
      />
    </el-select>
  </el-col>
  <el-col v-else :span="4">
    <el-input
      key="value"
      placeholder="填写属性值"
      :model-value="defaultValue"
      @change="inputChange"
    />
  </el-col>
</template>

<script lang="jsx">
export default {
  name: 'DevicePropertiesObject',
  components: {},
  props: {
    properties: {
      type: Object,
      default: null
    },
    actionData: {
      type: Object,
      default: null
    },
    propertiesData: {
      type: Object,
      default: null
    }
  },
  data() {
    return {
      defaultValue: ''
    }
  },
  created() {
    const propertiesData = this.propertiesData
    const properties = this.properties
    const actionData = this.actionData
    const message = actionData.configuration.message
    this.defaultValue =
      (message &&
        message.properties &&
        message.properties[propertiesData.id] &&
        message.properties[propertiesData.id][properties.id]) ||
      undefined
  },
  methods: {
    selectChange(value) {
      const propertiesData = this.propertiesData
      const properties = this.properties
      if (!this.actionData.configuration.message.properties[propertiesData.id]) {
        this.actionData.configuration.message.properties[propertiesData.id] = {}
      }
      this.actionData.configuration.message.properties[propertiesData.id][properties.id] = value
    },
    inputChange($event) {
      const propertiesData = this.propertiesData
      const properties = this.properties
      if (!this.actionData.configuration.message.properties[propertiesData.id]) {
        this.actionData.configuration.message.properties[propertiesData.id] = {}
      }
      this.actionData.configuration.message.properties[propertiesData.id][properties.id] =
        $event.target.value
    }
  }
}
</script>
