<template>
  <div>
    <el-form-item :label="label" :rules="rules">
      <el-select v-model="data.type" placeholder="请选择" @change="typeChange">
        <el-option-group label="基本类型">
          <el-option value="int">int(整数型)</el-option>
          <el-option value="long">long(长整数型)</el-option>
          <el-option value="float">float(单精度浮点型)</el-option>
          <el-option value="double">double(双精度浮点数)</el-option>
          <el-option value="string">text(字符串)</el-option>
          <el-option value="bool">bool(布尔型)</el-option>
        </el-option-group>
      </el-select>
    </el-form-item>
    <!-- -->
    <template v-if="['float', 'double'].indexOf(data.type) !== -1">
      <NumberItem :data="data" />
    </template>
    <template v-else-if="['int', 'long'].indexOf(data.type) !== -1">
      <IntegerItem :data="data" />
    </template>
    <template v-else-if="['string'].indexOf(data.type) !== -1">
      <StringItem :data="data" />
    </template>
    <template v-else-if="['bool'].indexOf(data.type) !== -1">
      <BooleanItem :data="data" />
    </template>
  </div>
</template>

<script lang="jsx">
import IntegerItem from './IntegerItem.vue'
import NumberItem from './NumberItem.vue'
import StringItem from './StringItem.vue'
import BooleanItem from './BooleanItem.vue'

export default {
  name: 'DataTypeItemSimple',
  components: {
    IntegerItem,
    NumberItem,
    StringItem,
    BooleanItem
  },
  props: {
    label: {
      type: String,
      default: ''
    },
    data: {
      type: Object,
      default: () => {}
    },
    rules: {
      type: Array,
      default: () => []
    }
  },
  data() {
    return {}
  },
  created() {},
  mounted() {},
  methods: {
    typeChange(value) {
      if (['float', 'double'].indexOf(value) !== -1) {
        this.setValue('scale', null)
        this.setValue('unit', null)
      } else if (['int', 'long'].indexOf(value) !== -1) {
        this.setValue('unit', null)
      } else if (['string'].indexOf(value) !== -1) {
        this.setValue('max', 64)
      } else if (['bool'].indexOf(value) !== -1) {
        this.setValue('trueText', null)
        this.setValue('trueValue', null)
        this.setValue('falseText', null)
        this.setValue('falseValue', null)
      } else {
        this.$message.error('不支持类型' + value)
      }
      console.log(value)
    },
    setValue(key, value) {
      this.data[key] = value
    }
  }
}
</script>
