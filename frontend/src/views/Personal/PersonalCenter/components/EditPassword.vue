<script setup lang="ts">
import { Form, FormSchema } from '@/components/Form'
import { useForm } from '@/hooks/web/useForm'
import { reactive, ref } from 'vue'
import { useValidator } from '@/hooks/web/useValidator'
import { ElMessage, ElMessageBox, ElDivider } from 'element-plus'
import { updatePwd } from '@/views/sys/api.js'

const { required } = useValidator()

const formSchema = reactive<FormSchema[]>([
  {
    field: 'passwordOld',
    label: '旧密码',
    component: 'InputPassword',
    colProps: {
      span: 24
    }
  },
  {
    field: 'password',
    label: '新密码',
    component: 'InputPassword',
    colProps: {
      span: 24
    },
    componentProps: {
      strength: true
    }
  },
  {
    field: 'password2',
    label: '确认新密码',
    component: 'InputPassword',
    colProps: {
      span: 24
    },
    componentProps: {
      strength: true
    }
  }
])

const rules = reactive({
  passwordOld: [required()],
  password: [
    required(),
    {
      asyncValidator: async (_, val, callback) => {
        const formData = await getFormData()
        const { password2 } = formData
        if (val !== password2) {
          callback(new Error('新密码与确认新密码不一致'))
        } else {
          callback()
        }
      }
    }
  ],
  password2: [
    required(),
    {
      asyncValidator: async (_, val, callback) => {
        const formData = await getFormData()
        const { password } = formData
        if (val !== password) {
          callback(new Error('确认新密码与新密码不一致'))
        } else {
          callback()
        }
      }
    }
  ]
})

const { formRegister, formMethods } = useForm()
const { getFormData, getElFormExpose } = formMethods

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
        updatePwd(data)
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
  <BaseButton type="primary" @click="save">确认修改</BaseButton>
</template>
