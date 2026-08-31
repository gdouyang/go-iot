<template>
  <div>
    <div class="table-page-search-wrapper">
      <el-form layout="inline">
        <el-row :gutter="48">
          <el-col :md="8" :sm="24">
            <el-form-item label="产品">
              <el-select
                v-model="searchObj.productId"
                placeholder="请选择产品"
                filterable
                clearable
                @change="search"
              >
                <el-option v-for="p in productList" :key="p.id" :value="p.id" :label="p.name" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :md="8" :sm="24">
            <el-form-item label="文件名">
              <el-input v-model="searchObj.name" placeholder="请输入文件名" @keyup.enter="search" />
            </el-form-item>
          </el-col>
          <el-col :md="6" :sm="24">
            <span class="table-page-search-submitButtons">
              <el-button type="primary" @click="search">查询</el-button>
              <el-button style="margin-left: 8px" @click="resetSearch">重置</el-button>
            </span>
          </el-col>
        </el-row>
      </el-form>
    </div>
    <div class="table-operator">
      <el-button type="primary" style="margin-left: 8px" @click="showAddDialog">新增</el-button>
      <el-button type="danger" style="margin-left: 8px" @click="handleBatchDelete"
        >批量删除</el-button
      >
    </div>
    <PageTable ref="tb" :url="tableUrl" :columns="columns" rowKey="id" />

    <!-- Add/Update Dialog -->
    <el-dialog v-model="dialogVisible" :title="dialogTitle" width="500px">
      <el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
        <el-form-item label="产品" prop="productId">
          <el-select v-model="form.productId" placeholder="请选择产品" filterable>
            <el-option v-for="p in productList" :key="p.id" :value="p.id" :label="p.name" />
          </el-select>
        </el-form-item>
        <el-form-item
          v-if="!form.id"
          label="文件"
          prop="file"
          :rules="[{ required: true, message: '请选择文件', trigger: 'change' }]"
        >
          <input type="file" @change="handleFileChange" />
        </el-form-item>
        <el-form-item label="名称" prop="name">
          <el-input v-model="form.name" placeholder="请输入名称" />
        </el-form-item>
      </el-form>
      <template #footer>
        <span class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button type="primary" @click="handleSubmit">确定</el-button>
        </span>
      </template>
    </el-dialog>
  </div>
</template>

<script lang="jsx" setup>
import { ref, reactive, nextTick, onMounted } from 'vue'
import { otaFilePageUrl, otaFileAdd, otaFileUpdate, otaFileDelete, getProductList } from '../api.js'
import { ElMessage, ElMessageBox } from 'element-plus'

const tableUrl = otaFilePageUrl
const searchObj = reactive({
  productId: '',
  name: ''
})
const tb = ref(null)
const productList = ref([])

onMounted(async () => {
  const res = await getProductList()
  if (res.success) {
    productList.value = res.result
  }
})

const columns = [
  { type: 'selection', width: 55, align: 'center' },
  { label: 'ID', field: 'id' },
  { label: '产品ID', field: 'productId' },
  { label: '名称', field: 'name' },
  { label: '路径', field: 'path' },
  {
    label: '大小',
    field: 'size',
    slots: {
      default: (data) => {
        const size = data.row.size
        if (size < 1024) return size + ' B'
        if (size < 1024 * 1024) return (size / 1024).toFixed(2) + ' KB'
        return (size / 1024 / 1024).toFixed(2) + ' MB'
      }
    }
  },
  { label: '创建时间', field: 'createTime' },
  {
    label: '操作',
    field: 'action',
    slots: {
      default: (data) => {
        return (
          <>
            <el-button link type="primary" onClick={() => handleEdit(data.row)}>
              编辑
            </el-button>
            <el-button link type="danger" onClick={() => handleDelete(data.row)}>
              删除
            </el-button>
          </>
        )
      }
    }
  }
]

const search = () => {
  const condition = []
  if (searchObj.productId) {
    condition.push({ key: 'productId', value: searchObj.productId, oper: 'EQ' })
  }
  if (searchObj.name) {
    condition.push({ key: 'name', value: searchObj.name, oper: 'LIKE' })
  }
  tb.value?.search(condition)
}

const resetSearch = () => {
  searchObj.productId = ''
  searchObj.name = ''
  search()
}

onMounted(() => {
  search()
})

// Dialog logic
const dialogVisible = ref(false)
const dialogTitle = ref('新增媒体')
const formRef = ref(null)
const form = reactive({
  id: '',
  name: '',
  productId: '',
  file: null
})
const rules = {
  productId: [{ required: true, message: '请选择产品', trigger: 'change' }],
  name: [{ required: true, message: '请输入名称', trigger: 'blur' }]
}

const showAddDialog = () => {
  dialogTitle.value = '新增媒体'
  form.id = ''
  form.productId = ''
  form.name = ''
  form.file = null
  dialogVisible.value = true
  nextTick(() => {
    formRef.value?.clearValidate()
  })
}

const handleEdit = (row) => {
  dialogTitle.value = '编辑媒体'
  form.id = row.id
  form.productId = row.productId
  form.name = row.name
  form.file = null
  dialogVisible.value = true
  nextTick(() => {
    formRef.value?.clearValidate()
  })
}

const handleFileChange = (e) => {
  const files = e.target.files
  if (files && files.length > 0) {
    form.file = files[0]
    if (!form.name) {
      form.name = files[0].name
    }
  }
}

const handleSubmit = () => {
  formRef.value.validate(async (valid) => {
    if (valid) {
      const formData = new FormData()
      formData.append('productId', form.productId)
      formData.append('name', form.name)
      if (form.file) {
        formData.append('file', form.file)
      }
      if (form.id) {
        formData.append('id', form.id)
        await otaFileUpdate(formData)
        ElMessage.success('更新成功')
      } else {
        if (!form.file) {
          ElMessage.error('请选择文件')
          return
        }
        await otaFileAdd(formData)
        ElMessage.success('新增成功')
      }
      dialogVisible.value = false
      search()
    }
  })
}

const handleDelete = (row) => {
  ElMessageBox.confirm('确认删除该记录吗？', '提示', {
    type: 'warning'
  }).then(async () => {
    await otaFileDelete([row.id])
    ElMessage.success('删除成功')
    search()
  })
}

const handleBatchDelete = () => {
  const ids = tb.value?.getSelectedRows('id')
  if (!ids || ids.length === 0) {
    ElMessage.warning('请选择要删除的记录')
    return
  }
  ElMessageBox.confirm(`确认删除选中的 ${ids.length} 条记录吗？`, '提示', {
    type: 'warning'
  }).then(async () => {
    await otaFileDelete(ids)
    ElMessage.success('批量删除成功')
    search()
  })
}
</script>
