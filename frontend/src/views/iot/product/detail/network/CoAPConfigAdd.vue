<template>
  <div>
    <Dialog
      ref="addModal"
      title="Http配置"
      width="510"
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
        <el-form-item label="协议路由">
          <el-card shadow="never">
            <el-row v-for="(i, index) in addObj.configuration.routers" :key="index" class="mb-5px">
              <el-col span="15">
                <el-input v-model="i.url" placeholder="/**" />
              </el-col>
              <el-col span="8" class="ml-10px text-center">
                <el-button
                  :size="'small'"
                  @click="minusRouter(index)"
                  :disabled="index === 0"
                  class="btn"
                >
                  -
                </el-button>
                <el-divider direction="vertical" />
                <el-button :size="'small'" @click="plusRouter" class="btn">+</el-button>
              </el-col>
            </el-row>
          </el-card>
        </el-form-item>
      </el-form>
    </Dialog>
  </div>
</template>

<script lang="jsx">
// import dayjs from 'dayjs'
import _ from 'lodash-es'
import { newHttpAddObj } from './entity.js'
import Base from './Base.vue'

export default {
  name: 'WebSocketConfigAdd',
  components: {},
  mixins: [Base],
  props: {},
  data() {
    return {
      addObj: newHttpAddObj(),
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
      this.getNetwork(productId, newHttpAddObj()).then((data) => {
        this.isEdit = false
        if (data.id) {
          this.isEdit = true
        }
        this.addObj = data
        this.$refs.addModal.open()
      })
    },
    addClose() {
      this.addObj = newHttpAddObj()
      this.$refs.addFormRef.clearValidate()
    },
    addConfirm() {
      this.$refs.addFormRef.validate((valid) => {
        if (valid) {
          const saveData = _.cloneDeep(this.addObj)
          let urlDup = false
          _.forEach(saveData.configuration.routers, (r) => {
            const find = _.find(
              saveData.configuration.routers,
              (p) => p.id !== r.id && p.url === r.url
            )
            if (find) {
              urlDup = true
            }
          })
          if (urlDup) {
            this.$message.error('路由重复，请修改')
            return
          }
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
    },
    minusRouter(index) {
      this.addObj.configuration.routers.splice(index, 1)
    },
    plusRouter() {
      this.addObj.configuration.routers.push({
        url: ''
      })
    }
  }
}
</script>

<style lang="less" scoped>
.btn {
  height: 32px;
  width: 32px;
}
</style>
