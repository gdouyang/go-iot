<template>
  <Dialog
    ref="addModal"
    :width="500"
    maxHeight="auto"
    title="批量导入设备"
    :okBtnLoading="okBtnLoading"
    @confirm="addConfirm"
    @close="addClose"
  >
    <el-form ref="addFormRef" :model="addObj" label-width="auto">
      <el-form-item
        label="产品"
        prop="productId"
        :rules="[{ required: true, message: '请选择产品', trigger: 'blur' }]"
      >
        <el-select v-model="addObj.productId" placeholder="产品" @change="productIdChange">
          <el-option v-for="p in productList" :key="p.id" :value="p.id">{{ p.name }}</el-option>
        </el-select>
      </el-form-item>
      <el-form-item
        v-show="addObj.productId"
        label="文件"
        prop="fileName"
        :rules="[{ required: true, message: '请选择文件', trigger: 'blur' }]"
      >
        <el-upload
          name="file"
          accept=".xlsx"
          :multiple="false"
          :show-file-list="false"
          :with-credentials="true"
          :before-upload="beforeUpload"
          @change="uploadChange"
        >
          <BaseButton>
            <template #icon><Icon icon="el:Upload" /></template>
            选择文件
          </BaseButton>
        </el-upload>
        <span>{{ addObj.fileName }}</span>
      </el-form-item>
      <div v-if="addObj.productId">
        <el-button link type="primary" :href="templateDownloadUrl">下载模版</el-button>
      </div>
      <div v-if="importLoading">
        <el-result
          :icon="isFinish ? 'success' : 'info'"
          :title="isFinish ? '已完成' : '进行中'"
          :sub-title="`总数量 ${count}`"
        >
          <template #extra>
            <p style="color: red">{{ errMessage }}</p>
          </template>
        </el-result>
      </div>
    </el-form>
  </Dialog>
</template>

<script lang="jsx">
import { getProductList, getTemplateDownloadUrl, getImportResultUrl, importDevice } from '../api'
import _ from 'lodash-es'
const defaultAddObj = {
  productId: undefined,
  fileName: undefined
}
export default {
  name: 'DeviceImport',
  data() {
    return {
      addObj: _.cloneDeep(defaultAddObj),
      productList: [],
      importLoading: false,
      okBtnLoading: false,
      isFinish: false,
      count: 0,
      errMessage: null,
      templateDownloadUrl: ''
    }
  },
  created() {},
  methods: {
    open() {
      this.listAllProduct().then(() => {
        this.$refs.addModal.open()
      })
    },
    uploadChange(data) {
      this.addObj.fileName = data.name
    },
    beforeUpload(file) {
      this.file = file
      return false
    },
    addClose() {
      this.importLoading = false
      this.okBtnLoading = false
      this.isFinish = false
      this.count = 0
      this.errMessage = null
      this.addObj = _.cloneDeep(defaultAddObj)
      this.$refs.addFormRef.clearValidate()
    },
    addConfirm() {
      if (this.okBtnLoading) {
        return
      }
      this.$refs.addFormRef.validate((valid) => {
        if (valid) {
          this.importLoading = true
          this.okBtnLoading = true
          importDevice(this.addObj.productId, this.file).then((resp) => {
            if (resp.success) {
              this.showImportResult(resp.result)
            } else {
              this.$message.success(resp.message)
              this.okBtnLoading = true
            }
          })
        }
      })
    },
    showImportResult(id) {
      var source = new EventSource(getImportResultUrl(id))
      source.onmessage = (e) => {
        const res = JSON.parse(e.data)
        if (res.success) {
          const temp = res.result.num
          this.count = temp
          if (res.result.finish) {
            this.isFinish = true
            this.okBtnLoading = false
            source.close()
            this.$message.success('操作成功')
            this.$refs.addModal.close()
            this.$emit('success')
          }
        } else {
          this.isFinish = true
          this.okBtnLoading = false
          this.errMessage = res.message
        }
      }
      source.onerror = () => {
        // this.isFinish = true
        source.close()
        // this.$refs.addModal.close()
        // this.$emit('success')
      }
      // 添加一个开启回调
      source.onopen = function (event) {}
    },
    listAllProduct() {
      return getProductList().then((resp) => {
        if (resp.success) {
          this.productList = resp.result
        }
      })
    },
    productIdChange() {
      this.templateDownloadUrl = getTemplateDownloadUrl(this.addObj.productId)
    }
  }
}
</script>

<style lang="less"></style>
