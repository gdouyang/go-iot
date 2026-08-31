<template>
  <div>
    <Dialog ref="addModal" @confirm="handleOk" @close="handleCancel" :width="500" maxHeight="auto">
      <el-form ref="addFormRef" :model="addObj" label-width="auto">
        <el-form-item label="端口" prop="port" :rules="[{ required: true, message: '请输入' }]">
          <el-input-number
            v-model.number="addObj.port"
            placeholder="端口"
            :max="65535"
            controls-position="right"
          />
        </el-form-item>
      </el-form>
    </Dialog>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { getNetwork, editNetwork, addNetwork } from '../networkapi.js'
const defaultAddObj = {
  port: null,
  productId: '',
  configuration: '',
  type: '',
  state: ''
}
export default {
  name: 'NetworkModal',
  components: {},
  data() {
    return {
      addObj: _.cloneDeep(defaultAddObj),
      spinning: 0,
      isEdit: false
    }
  },
  created() {},
  methods: {
    add() {
      this.addObj = _.cloneDeep(defaultAddObj)
      this.isEdit = false
      this.$refs.addModal.open({ title: '新增' })
    },
    edit(record) {
      this.isEdit = true
      this.spinning++
      getNetwork(record.id)
        .then((data1) => {
          if (data1.success) {
            const data = data1.result
            this.$refs.addModal.open({ title: '修改' })
            this.addObj = data
          }
        })
        .finally(() => {
          this.spinning--
        })
    },
    close() {
      this.$refs.addModal.close()
    },
    handleOk() {
      const _this = this
      this.$refs.addFormRef.validate((valid) => {
        if (valid) {
          const values = _.cloneDeep(this.addObj)
          console.log('form values', values)
          let promise = null
          if (this.isEdit) {
            promise = editNetwork(values)
          } else {
            promise = addNetwork(values)
          }
          promise.then((data) => {
            if (data.success) {
              this.$message.success('操作成功')
              _this.close()
              this.$emit('ok')
            } else {
              this.$message.success(data.message)
            }
          })
        }
      })
    },
    handleCancel() {
      this.$refs.addFormRef.clearValidate()
    }
  }
}
</script>

<style scoped></style>
