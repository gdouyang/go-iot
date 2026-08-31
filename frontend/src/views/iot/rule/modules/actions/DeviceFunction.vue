<template>
  <el-col v-if="item.type === 'enum'" :span="6">
    <el-select placeholder="选择调用参数" v-model="defaultValue" @change="inputsChange">
      <el-option
        v-for="_item in item.elements"
        :key="_item.value"
        :value="_item.value"
        :label="`${_item.text}（${_item.value}）`"
      />
    </el-select>
  </el-col>
  <el-col v-else-if="item.type === 'bool'" :span="6">
    <el-select placeholder="选择调用参数" v-model="defaultValue" @change="inputsChange">
      <el-option :value="item.trueValue + ''" :label="`${item.trueText}（${item.trueValue}）`" />
      <el-option :value="item.falseValue + ''" :label="`${item.falseText}（${item.falseValue}）`" />
    </el-select>
  </el-col>
  <el-col
    v-else-if="
      item.type === 'int' || item.type === 'long' || item.type === 'float' || item.type === 'double'
    "
    :span="6"
  >
    <el-input-number v-model="defaultValue" controls-position="right" @change="inputsChange" />
  </el-col>
  <el-col v-else-if="item.type === 'password'" :span="6">
    <el-input
      type="password"
      show-password
      v-model="defaultValue"
      :maxlength="100"
      @change="inputsChange"
    />
  </el-col>
  <el-col v-else :span="6">
    <el-input placeholder="填写调用参数" v-model="defaultValue" @change="inputsChange" />
  </el-col>
</template>

<script lang="jsx">
export default {
  name: 'DeviceFunction',
  components: {},
  props: {
    item: {
      type: Object,
      default: null
    },
    index: {
      type: Number,
      default: null
    },
    actionData: {
      type: Object,
      default: null
    }
  },
  data() {
    return {
      defaultValue: null
    }
  },
  computed: {},
  created() {
    const message = this.actionData.configuration
    const item = this.item
    if (item && message && message.data) {
      this.defaultValue = message.data[item.id]
    } else {
      this.defaultValue = null
    }
  },
  methods: {
    inputsChange() {
      const item = this.item
      this.actionData.configuration.data = {}
      this.actionData.configuration.data[item.id] = this.defaultValue
    }
  }
}
</script>
