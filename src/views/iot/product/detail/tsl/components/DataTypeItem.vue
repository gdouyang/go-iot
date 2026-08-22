<template>
  <div>
    <el-form-item :label="label" :rules="rules">
      <el-select v-model="data.type" placeholder="请选择" @change="typeChange">
        <el-option-group label="基本类型">
          <el-option value="int" label="int(整数型)" />
          <el-option value="long" label="long(长整数型)" />
          <el-option value="float" label="float(单精度浮点型)" />
          <el-option value="double" label="double(双精度浮点数)" />
          <el-option value="string" label="text(字符串)" />
          <el-option value="bool" label="bool(布尔型)" />
        </el-option-group>
        <el-option-group label="其他类型" v-if="showOtherGroup">
          <el-option value="date" label="date(时间型)" />
          <el-option value="enum" label="enum(枚举)" />
          <!-- <el-option value="array">array(数组)</el-option> -->
          <el-option value="object" label="object(结构体)" />
          <!-- <el-option value="file">file(文件)</el-option> -->
          <el-option value="password" label="password(密码)" />
          <!-- <el-option value="geoPoint">geoPoint(地理位置)</el-option> -->
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
    <template v-else-if="['date'].indexOf(data.type) !== -1">
      <DateItem :data="data" />
    </template>
    <template v-else-if="['enum'].indexOf(data.type) !== -1">
      <EnumItem :data="data" />
    </template>
    <template v-else-if="['password'].indexOf(data.type) !== -1">
      <PasswordItem :data="data" />
    </template>
    <template v-else-if="['file'].indexOf(data.type) !== -1">
      <FileItem :data="data" />
    </template>
    <template v-else-if="['array'].indexOf(data.type) !== -1">
      <DataTypeItemSimple
        label="元素类型"
        :data="data"
        field="elementType"
        :showOtherGroup="false"
        :prop="field + '.elementType.type'"
        :rules="[{ required: true, message: '请选择' }]"
      />
    </template>
    <template v-else-if="['object'].indexOf(data.type) !== -1">
      <ObjectItem :data="data" />
    </template>
  </div>
</template>

<script lang="jsx">
import IntegerItem from './IntegerItem.vue'
import NumberItem from './NumberItem.vue'
import StringItem from './StringItem.vue'
import BooleanItem from './BooleanItem.vue'
import DateItem from './DateItem.vue'
import EnumItem from './EnumItem.vue'
import PasswordItem from './PasswordItem.vue'
import FileItem from './FileItem.vue'
import ObjectItem from './ObjectItem.vue'
import DataTypeItemSimple from './DataTypeItemSimple.vue'

export default {
  name: 'DataTypeItem',
  components: {
    IntegerItem,
    NumberItem,
    StringItem,
    BooleanItem,
    DateItem,
    EnumItem,
    PasswordItem,
    FileItem,
    ObjectItem,
    DataTypeItemSimple
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
    showOtherGroup: {
      type: Boolean,
      default: true
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
      } else if (['string', 'password'].indexOf(value) !== -1) {
        this.setValue('max', 64)
      } else if (['bool'].indexOf(value) !== -1) {
        this.setValue('trueText', null)
        this.setValue('trueValue', null)
        this.setValue('falseText', null)
        this.setValue('falseValue', null)
      } else if (['date'].indexOf(value) !== -1) {
        this.setValue('format', null)
      } else if (['enum'].indexOf(value) !== -1) {
        this.setValue('elements', [{ text: '', value: '' }])
      } else if (['file'].indexOf(value) !== -1) {
        this.setValue('bodyType', 'url') // url, base64
      } else if (['array'].indexOf(value) !== -1) {
        this.setValue('elementType', { type: null })
      } else if (['object'].indexOf(value) !== -1) {
        this.setValue('properties', [])
      }
      console.log(value)
    },
    setValue(key, value) {
      this.data[key] = value
      // this.$set(this.data, key, value)
    }
  }
}
</script>
