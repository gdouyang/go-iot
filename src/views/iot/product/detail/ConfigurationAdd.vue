<template>
  <el-drawer
    :title="isEdit ? '修改配置' : '添加配置'"
    width="500"
    :model-value="true"
    :close-on-click-modal="false"
    @close="visibleChange(false)"
  >
    <el-form ref="addFormRef" :model="configuration" label-width="auto">
      <el-row :gutter="16">
        <el-col>
          <el-form-item
            prop="property"
            label="Key"
            :rules="[
              { required: true, message: 'Key不能为空', trigger: 'blur' },
              {
                pattern: new RegExp(/^[0-9a-zA-Z]+$/, 'g'),
                message: 'key只能由数字、字母组成',
                trigger: 'blur'
              }
            ]"
          >
            <el-input v-model="configuration.property" :disabled="isEdit" :maxlength="32" />
          </el-form-item>
          <el-form-item
            prop="type"
            label="类型"
            :rules="[{ required: true, message: '类型不能为空', trigger: 'change' }]"
          >
            <el-select v-model="configuration.type" :disabled="configuration.buildin">
              <el-option value="string">字符串</el-option>
              <el-option value="number">数字</el-option>
              <el-option value="password">密码</el-option>
            </el-select>
          </el-form-item>
          <el-form-item label="值">
            <el-input
              v-if="configuration.type === 'password'"
              type="password"
              show-password
              v-model="configuration.value"
              :maxlength="100"
            />
            <el-input v-model="configuration.value" :maxlength="100" v-else />
          </el-form-item>
          <el-form-item label="描述">
            <el-input
              v-model="configuration.desc"
              :disabled="configuration.buildin"
              :maxlength="100"
            />
          </el-form-item>
        </el-col>
      </el-row>
    </el-form>
    <div class="drawer-footer">
      <el-button style="margin-right: 8px" @click="visibleChange(false)">关闭</el-button>
      <el-button type="primary" @click="saveData">保存</el-button>
    </div>
  </el-drawer>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { updateProduct } from '@/views/iot/product/api.js'
const defaultData = { property: '', desc: '', type: null, value: '' }
export default {
  name: 'ConfigurationAdd',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    allConfig: {
      type: Array,
      default: () => []
    },
    productId: {
      type: String,
      default: () => null
    },
    data: {
      type: Object,
      default: () => null
    }
  },
  data() {
    return {
      openFlag: false,
      configuration: _.cloneDeep(defaultData),
      isEdit: false
    }
  },
  watch: {
    visible(newVal) {
      this.openFlag = newVal
    }
  },
  mounted() {
    this.visibleChange(true)
  },
  methods: {
    visibleChange(flag) {
      if (!flag) {
        this.$emit('close')
      } else {
        if (this.data) {
          const d = _.cloneDeep(this.data)
          this.configuration = d
          this.isEdit = true
        } else {
          this.isEdit = false
          this.configuration = _.cloneDeep(defaultData)
        }
      }
    },
    saveData() {
      this.$refs.addFormRef.validate((valid, object) => {
        if (valid) {
          const p = _.cloneDeep(this.configuration)
          const config1 = _.filter(this.allConfig, (c) => c.property === p.property)
          if ((!this.isEdit && _.size(config1) > 0) || (this.isEdit && _.size(config1) > 1)) {
            this.$message.error(p.property + '已经存在')
            return
          }
          if (this.isEdit) {
            const conf = _.cloneDeep(this.allConfig)
            const find = _.find(conf, (c) => c.property === p.property)
            find.value = p.value
            find.desc = p.desc
            find.type = p.type
            const param = {
              id: this.productId,
              metaconfig: conf
            }
            this.updateData(param)
          } else {
            const conf = _.cloneDeep(this.allConfig)
            conf.push(p)
            const param = {
              id: this.productId,
              metaconfig: conf
            }
            this.updateData(param)
          }
        }
      })
    },
    updateData(item) {
      const param = {
        id: this.productId,
        metaconfig: item.metaconfig
      }
      updateProduct(this.productId, param).then((resp) => {
        if (resp.success) {
          this.$message.success('配置信息修改成功')
          this.visibleChange(false)
        }
      })
    }
  }
}
</script>

<style lang="less" scoped></style>
