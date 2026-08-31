<script lang="ts" setup>
import { FormSchema, Form } from '@/components/Form'
import { useForm } from '@/hooks/web/useForm'
import { useValidator } from '@/hooks/web/useValidator'
import { reactive, ref, watch } from 'vue'
import { ElDivider, ElMessage, ElMessageBox } from 'element-plus'
import { saveUserInfo } from '@/views/sys/api.js'

const props = defineProps({
  userInfo: {
    type: Object,
    default: () => ({})
  }
})

const { required, maxlength, email } = useValidator()

const formSchema = reactive<FormSchema[]>([
  {
    field: 'nickname',
    label: '昵称',
    component: 'Input',
    colProps: {
      span: 24
    }
  },
  {
    field: 'email',
    label: '邮箱',
    component: 'Input',
    colProps: {
      span: 24
    }
  }
])

const rules = reactive({
  nickname: [required(), maxlength(50)],
  // phoneNumber: [phone()],
  email: [email()]
})

const { formRegister, formMethods } = useForm()
const { setValues, getElFormExpose } = formMethods

watch(
  () => props.userInfo,
  (value) => {
    setValues(value)
  },
  {
    immediate: true,
    deep: true
  }
)

const saveLoading = ref(false)
const save = async () => {
  const elForm = await getElFormExpose()
  const valid = await elForm?.validate().catch((err) => {
    console.log(err)
  })
  if (valid) {
    ElMessageBox.confirm('是否确认修改?', '提示', {
      confirmButtonText: '确认',
      cancelButtonText: '取消',
      type: 'warning'
    })
      .then(async () => {
        saveLoading.value = true
        const data = await formMethods.getFormData()
        await saveUserInfo(data)
          .then(() => {
            ElMessage.success('修改成功')
          })
          .finally(() => {
            saveLoading.value = false
          })
      })
      .catch(() => {})
  }
}
</script>

<template>
  <Form :rules="rules" @register="formRegister" :schema="formSchema" />
  <ElDivider />
  <BaseButton type="primary" @click="save">保存</BaseButton>
</template>
