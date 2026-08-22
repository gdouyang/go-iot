<template>
  <el-form-item
    label="枚举项"
    required
    :rules="[{ validator: validateElements, trigger: ['blur', 'change'] }]"
  >
    <div style="width: 100%">
      <div v-for="(item, index) in arrayEnumData" :key="index" class="enum-row">
        <el-row :gutter="8" style="margin-bottom: 8px">
          <el-col :span="10">
            <el-input placeholder="标识 (如: 1 或 open)" v-model="item.value" clearable />
          </el-col>
          <el-col :span="2" style="text-align: center; line-height: 32px">
            <Icon icon="el:ArrowRight" />
          </el-col>
          <el-col :span="10">
            <el-input placeholder="描述 (如: 开启)" v-model="item.text" clearable />
          </el-col>
          <el-col :span="2" style="text-align: center; line-height: 32px">
            <div class="flex items-center justify-center h-full">
              <Icon
                v-if="index === arrayEnumData.length - 1"
                icon="el:CirclePlus"
                title="添加枚举项"
                class="cursor-pointer"
                @click="plus"
              />
              <Icon
                v-if="arrayEnumData.length > 1"
                icon="el:Remove"
                title="删除枚举项"
                class="cursor-pointer"
                style="margin-left: 8px"
                @click="minus(index)"
              />
            </div>
          </el-col>
        </el-row>
      </div>
    </div>
  </el-form-item>
</template>

<script lang="jsx">
import _ from 'lodash-es'
export default {
  name: 'EnumItem',
  components: {},
  props: {
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      arrayEnumData: []
    }
  },
  watch: {
    'data.elements': {
      handler(val) {
        if (Array.isArray(val) && val.length > 0) {
          this.arrayEnumData = val
        } else {
          this.arrayEnumData = [{ text: '', value: '' }]
          this.data.elements = this.arrayEnumData
        }
      },
      immediate: true
    }
  },
  methods: {
    plus() {
      this.arrayEnumData.push({ text: '', value: '' })
    },
    minus(index) {
      this.arrayEnumData.splice(index, 1)
    },
    validateElements(rule, value, callback) {
      if (!this.arrayEnumData || !this.arrayEnumData.length) {
        return callback(new Error('请至少添加一个枚举项'))
      }
      const values = new Set()
      for (let i = 0; i < this.arrayEnumData.length; i++) {
        const item = this.arrayEnumData[i]
        const valStr = item.value != null ? String(item.value).trim() : ''
        const textStr = item.text != null ? String(item.text).trim() : ''
        if (!valStr) {
          return callback(new Error(`第 ${i + 1} 个枚举项的标识不能为空`))
        }
        if (!textStr) {
          return callback(new Error(`第 ${i + 1} 个枚举项的描述不能为空`))
        }
        if (values.has(valStr)) {
          return callback(new Error(`枚举项标识 "${valStr}" 重复，请修改`))
        }
        values.add(valStr)
      }
      callback()
    }
  }
}
</script>
<style lang="less" scoped>
.enum-row {
  width: 100%;
  .el-icon {
    cursor: pointer;
  }
}
</style>
