<template>
  <Dialog
    v-model="visible"
    title="OTA升级"
    :width="600"
    :showOk="true"
    @close="visible = false"
    @confirm="handleUpdate"
  >
    <el-form ref="formRef" :model="otaForm" :rules="rules" label-width="120px">
      <el-form-item label="产品" prop="productId">
        <el-select
          v-model="otaForm.productId"
          placeholder="请选择产品"
          filterable
          @change="handleProductChange"
        >
          <el-option v-for="p in productList" :key="p.id" :value="p.id" :label="p.name" />
        </el-select>
      </el-form-item>
      <el-form-item label="设备ID列表" prop="deviceIds">
        <div style="margin-bottom: 8px">
          <el-tag
            v-for="id in otaForm.deviceIds"
            :key="id"
            closable
            style="margin-right: 5px; margin-bottom: 5px"
            @close="handleCloseTag(id)"
          >
            {{ id }}
          </el-tag>
        </div>
        <el-button type="primary" @click="openDeviceSelect" :disabled="!otaForm.productId"
          >选择设备</el-button
        >
        <DeviceSelect
          ref="deviceSelectRef"
          :productId="otaForm.productId"
          :multiple="true"
          @select-all="handleDeviceSelectAll"
        />
      </el-form-item>
      <el-form-item label="选择OTA包" prop="otaFileId">
        <el-select v-model="otaForm.otaFileId" placeholder="请选择OTA包" style="width: 100%">
          <el-option
            v-for="item in mediaList"
            :key="item.id"
            :value="item.id"
            :label="item.name + ' (' + formatSize(item.size) + ')'"
          />
        </el-select>
      </el-form-item>
      <el-form-item label="分块大小" prop="chunkSize">
        <el-input-number v-model="otaForm.chunkSize" :min="1" :step="1024" />
        <span style="margin-left: 10px">字节 (默认 1024)</span>
      </el-form-item>
      <el-form-item label="超时时间" prop="timeout">
        <el-input-number v-model="otaForm.timeout" :min="1" />
        <span style="margin-left: 10px">秒 (默认 10)</span>
      </el-form-item>
    </el-form>
  </Dialog>
</template>

<script setup>
import { ref, nextTick, watch } from 'vue'
import { otaUpdate, getProductList as getProductListApi, otaFileList } from '../api.js'
import { ElMessage } from 'element-plus'
import DeviceSelect from '../DeviceSelect.vue'

const emit = defineEmits(['success'])

const visible = ref(false)
const productList = ref([])
const mediaList = ref([])
const formRef = ref(null)
const fileInput = ref(null)
const deviceSelectRef = ref(null)
const uploadMode = ref('select')

const otaForm = ref({
  productId: '',
  deviceIds: [],
  file: null,
  otaFileId: '',
  chunkSize: 1024,
  timeout: 10
})

const rules = {
  productId: [{ required: true, message: '请选择产品', trigger: 'change' }],
  deviceIds: [{ required: true, message: '请输入设备ID', trigger: 'change', type: 'array' }],
  file: [{ required: true, message: '请选择文件', trigger: 'change' }],
  otaFileId: [{ required: true, message: '请选择OTA包', trigger: 'change' }]
}

const formatSize = (size) => {
  if (size < 1024) return size + ' B'
  if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
  return (size / 1024 / 1024).toFixed(2) + ' MB'
}

const getProductList = async () => {
  const res = await getProductListApi()
  if (res.success) {
    productList.value = res.result
  }
}

const getMediaList = async () => {
  try {
    const res = await otaFileList(otaForm.value.productId)
    mediaList.value = res
  } catch (e) {
    console.error(e)
  }
}

const open = () => {
  visible.value = true
  getProductList()
  getMediaList()
  uploadMode.value = 'upload'
  otaForm.value = {
    productId: '',
    deviceIds: [],
    file: null,
    otaFileId: '',
    chunkSize: 1024,
    timeout: 10
  }
  if (fileInput.value) {
    fileInput.value.value = ''
  }
  nextTick(() => {
    if (formRef.value) {
      formRef.value.clearValidate()
    }
  })
}

const handleProductChange = () => {
  otaForm.value.deviceIds = []
  otaForm.value.otaFileId = ''
  getMediaList()
}

const openDeviceSelect = () => {
  deviceSelectRef.value.open()
}

const handleDeviceSelectAll = (items) => {
  items.forEach((item) => {
    if (!otaForm.value.deviceIds.includes(item.id)) {
      otaForm.value.deviceIds.push(item.id)
    }
  })
}

const handleCloseTag = (id) => {
  otaForm.value.deviceIds = otaForm.value.deviceIds.filter((item) => item !== id)
}

const handleFileChange = (e) => {
  const files = e.target.files
  if (files && files.length > 0) {
    otaForm.value.file = files[0]
    if (formRef.value) {
      formRef.value.validateField('file')
    }
  }
}

const handleUpdate = () => {
  formRef.value.validate(async (valid) => {
    if (valid) {
      if (uploadMode.value === 'select' && !otaForm.value.otaFileId) {
        ElMessage.error('请选择OTA包')
        return
      }
      try {
        await otaUpdate(
          otaForm.value.file,
          otaForm.value.deviceIds,
          otaForm.value.chunkSize,
          otaForm.value.timeout,
          otaForm.value.otaFileId
        )
        ElMessage.success('OTA升级已开始')
        visible.value = false
        emit('success')
      } catch (e) {
        // Error handling is typically done in axios interceptor
      }
    }
  })
}

defineExpose({
  open
})
</script>
