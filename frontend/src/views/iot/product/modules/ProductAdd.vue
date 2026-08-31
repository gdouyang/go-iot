<template>
  <div>
    <Dialog ref="addModal" @confirm="addConfirm" @close="addClose" maxHeight="auto">
      <el-form ref="addFormRef" :model="addObj" style="width: 90%" label-width="auto">
        <el-form-item
          label="产品ID"
          prop="id"
          :rules="[
            { required: true, message: '请输入产品ID' },
            { max: 32, message: '产品ID不超过32个字符' },
            {
              pattern: new RegExp(/^[0-9a-zA-Z_\-]+$/, 'g'),
              message: '产品ID只能由数字、字母、下划线、中划线组成'
            }
          ]"
        >
          <el-input
            v-model="addObj.id"
            :disabled="isEdit"
            placeholder="将自动转为大写"
            @input="onProductIdInput"
          />
          <div v-if="!isEdit" class="form-tip"> 创建后不可更改 </div>
        </el-form-item>
        <el-form-item
          label="名称"
          prop="name"
          :rules="[{ required: true, message: '名称不能为空' }]"
        >
          <el-input v-model="addObj.name" placeholder="名称" :maxlength="32" />
        </el-form-item>
        <el-form-item
          label="网络类型"
          prop="networkType"
          :rules="[{ required: true, message: '网络类型不能为空' }]"
        >
          <network-type-select v-model="addObj.networkType" :disabled="isEdit" />
          <div v-if="!isEdit" class="form-tip"> 创建后不可更改 </div>
        </el-form-item>
        <el-form-item
          label="时序存储"
          prop="storePolicy"
          :rules="[{ required: true, message: '请选择时序存储方式' }]"
        >
          <el-select
            v-model="addObj.storePolicy"
            placeholder="请选择"
            style="width: 100%"
            :disabled="isEdit"
          >
            <el-option
              v-for="item in storePolicyOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
          <div v-if="!isEdit" class="form-tip"> 创建后不可更改 </div>
        </el-form-item>
        <el-form-item
          v-if="addObj.storePolicy === 'es'"
          label="时序保留(月)"
          prop="retentionMonths"
        >
          <el-input-number
            v-model="addObj.retentionMonths"
            :min="0"
            :max="120"
            :step="1"
            :precision="0"
            :value-on-clear="0"
            controls-position="right"
            placeholder="0=跟随系统"
            style="width: 100%"
          />
          <div class="form-tip"> 0 跟随系统全局配置；大于 0 使用本产品的保留月数。 </div>
        </el-form-item>
        <el-form-item label="说明" prop="desc">
          <el-input
            type="textarea"
            v-model="addObj.desc"
            placeholder="说明"
            :maxlength="200"
            show-word-limit
          />
        </el-form-item>
      </el-form>
    </Dialog>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { get, addProduct, updateProduct, listStorePolicies } from '@/views/iot/product/api.js'
import NetworkTypeSelect from '../components/NetworkTypeSelect.vue'

const defaultAddObj = {
  id: null,
  name: '',
  metadata: { events: [], properties: [], functions: [] },
  desc: '',
  networkType: '',
  storePolicy: 'es',
  // 0 = 跟随系统全局；>0 = 产品配置
  retentionMonths: 0
}
export default {
  name: 'ProductAdd',
  components: {
    NetworkTypeSelect
  },
  data() {
    return {
      labelCol: {
        xs: { span: 24 },
        sm: { span: 5 }
      },
      wrapperCol: {
        xs: { span: 24 },
        sm: { span: 16 }
      },
      addObj: _.cloneDeep(defaultAddObj),
      isEdit: false,
      storePolicyOptions: [{ value: 'es', label: 'Elasticsearch' }],
      storePolicyDefault: 'es'
    }
  },
  created() {
    this.loadStorePolicies()
  },
  methods: {
    onProductIdInput(val) {
      if (this.isEdit) return
      const next = (val || '').toUpperCase()
      if (this.addObj.id !== next) {
        this.addObj.id = next
      }
    },
    loadStorePolicies() {
      listStorePolicies()
        .then((data) => {
          if (data && data.success && data.result) {
            const list = data.result.list
            if (Array.isArray(list) && list.length > 0) {
              this.storePolicyOptions = list
            }
            if (data.result.default) {
              this.storePolicyDefault = data.result.default
              if (!this.isEdit) {
                this.addObj.storePolicy = data.result.default
              }
            }
          }
        })
        .catch(() => {
          // 接口失败时保留本地默认 es/mock
        })
    },
    add() {
      this.isEdit = false
      this.addObj = _.cloneDeep(defaultAddObj)
      this.addObj.storePolicy = this.storePolicyDefault || 'es'
      this.loadStorePolicies()
      this.$refs.addModal.open({ title: '新增产品' })
    },
    edit(row) {
      this.isEdit = true
      get(row.id).then((data) => {
        if (data.success) {
          const result = data.result
          this.addObj.id = result.id
          this.addObj.name = result.name
          this.addObj.desc = result.desc
          this.addObj.networkType = result.networkType
          this.addObj.storePolicy = result.storePolicy || 'es'
          // 后端 null/缺省/0 → 0（跟随系统）；>0 → 产品配置
          const rm = result.retentionMonths
          this.addObj.retentionMonths =
            rm === null || rm === undefined || Number(rm) <= 0 ? 0 : Number(rm)
          // 编辑时若当前策略不在可选列表中（如 tdengine 已关闭），仍展示当前值
          if (
            this.addObj.storePolicy &&
            !this.storePolicyOptions.some((o) => o.value === this.addObj.storePolicy)
          ) {
            this.storePolicyOptions = [
              ...this.storePolicyOptions,
              {
                value: this.addObj.storePolicy,
                label: this.addObj.storePolicy + '（当前）'
              }
            ]
          }
          this.$refs.addModal.open({ title: '修改产品' })
        }
      })
    },
    addClose() {
      this.addObj = _.cloneDeep(defaultAddObj)
      this.addObj.storePolicy = this.storePolicyDefault || 'es'
      this.$refs.addFormRef.clearValidate()
    },
    /**
     * 规范化提交体：
     * retentionMonths：0=跟随系统；>0=产品配置。空/非法按 0 提交。
     */
    buildSaveData() {
      const saveData = _.cloneDeep(this.addObj)
      const v = saveData.retentionMonths
      if (v === null || v === undefined || v === '' || Number.isNaN(Number(v)) || Number(v) < 0) {
        saveData.retentionMonths = 0
      } else {
        saveData.retentionMonths = Math.floor(Number(v))
      }
      // 编辑不允许改 storePolicy，不传
      if (this.isEdit) {
        delete saveData.storePolicy
      }
      return saveData
    },
    addConfirm() {
      this.$refs.addFormRef.validate((valid) => {
        if (valid) {
          let promise = null
          const saveData = this.buildSaveData()
          if (this.isEdit) {
            delete saveData.metadata
            promise = updateProduct(saveData.id, saveData)
          } else {
            saveData.state = false
            saveData.metadata = JSON.stringify(this.addObj.metadata)
            promise = addProduct(saveData)
          }
          promise.then((resp) => {
            if (resp.success) {
              this.$message.success('操作成功')
              this.$refs.addModal.close()
              this.$emit('success')
            } else {
              this.$message.error(resp.message || '操作失败')
            }
          })
        }
      })
    }
  }
}
</script>

<style lang="less"></style>

<style lang="less" scoped>
.form-tip {
  margin-top: 4px;
  font-size: 12px;
  line-height: 1.4;
  color: var(--el-text-color-secondary);
}
</style>
