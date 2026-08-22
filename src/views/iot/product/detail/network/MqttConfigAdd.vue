<template>
  <div>
    <Dialog
      ref="addModal"
      title="MQTT配置"
      width="500"
      maxHeight="auto"
      @confirm="addConfirm"
      @close="addClose"
    >
      <el-form ref="addFormRef" :model="addObj" style="width: 90%" label-width="auto">
        <el-form-item label="开启SSL" prop="configuration.useTLS">
          <el-radio-group v-model="addObj.configuration.useTLS">
            <el-radio :value="true">是</el-radio>
            <el-radio :value="false">否</el-radio>
          </el-radio-group>
        </el-form-item>
        <el-form-item
          v-if="addObj.configuration.useTLS"
          label="crt文件"
          prop="certBase64"
          :rules="[{ required: true, message: 'crt文件不能为空', trigger: 'blur' }]"
        >
          <CertificateUpload v-model="addObj.certBase64" />
          <el-input type="textarea" v-model="addObj.certBase64" show-word-limit />
        </el-form-item>
        <el-form-item
          v-if="addObj.configuration.useTLS"
          label="key文件"
          prop="keyBase64"
          :rules="[{ required: true, message: 'key文件不能为空', trigger: 'blur' }]"
        >
          <CertificateUpload v-model="addObj.keyBase64" />
          <el-input type="textarea" v-model="addObj.keyBase64" show-word-limit />
        </el-form-item>
      </el-form>
    </Dialog>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { newMqttAddObj } from './entity.js'
import Base from './Base.vue'

export default {
  name: 'MqttConfigAdd',
  components: {},
  mixins: [Base],
  props: {},
  data() {
    return {
      addObj: newMqttAddObj(),
      isEdit: false
    }
  },
  computed: {},
  created() {},
  methods: {
    open(productId) {
      if (!productId) {
        this.$message.error('请指定产品ID')
        return
      }
      this.productId = productId
      this.getNetwork(productId, newMqttAddObj()).then((data) => {
        this.isEdit = false
        if (data.id) {
          this.isEdit = true
        }
        this.addObj = data
        this.$refs.addModal.open()
      })
    },
    addClose() {
      this.addObj = newMqttAddObj()
      this.$refs.addFormRef.clearValidate()
    },
    addConfirm() {
      this.$refs.addFormRef.validate((valid) => {
        if (valid) {
          const saveData = _.cloneDeep(this.addObj)
          this.updateNetwork(saveData).then((resp) => {
            if (resp.success) {
              this.$message.success('操作成功')
              this.$refs.addModal.close()
              this.$emit('success')
            } else {
              this.$message.error(resp.message)
            }
          })
        }
      })
    }
  }
}
</script>

<style lang="less" scoped></style>
