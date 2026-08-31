<template>
  <div>
    <Dialog ref="addModal" @confirm="handleOk" @close="handleCancel" :width="500" maxHeight="auto">
      <!-- v-loading 不能直接绑在 Dialog 上：其根节点是 ElDialog 组件，指令无法落到真实 DOM -->
      <div v-loading="loading">
        <el-form ref="addFormRef" :model="addObj" label-width="auto">
          <el-form-item
            label="账号"
            prop="username"
            :rules="[{ required: true, message: '请输入' }]"
          >
            <el-input v-model="addObj.username" :maxlength="32" show-word-limit />
          </el-form-item>
          <el-form-item
            label="名称"
            prop="nickname"
            :rules="[{ required: true, message: '请输入' }]"
          >
            <el-input v-model="addObj.nickname" :maxlength="32" show-word-limit />
          </el-form-item>

          <el-form-item
            label="状态"
            prop="enableFlag"
            :rules="[{ required: true, message: '请选择' }]"
          >
            <el-select v-model="addObj.enableFlag">
              <el-option value="true" label="启动" />
              <el-option value="false" label="禁用" />
            </el-select>
          </el-form-item>

          <el-form-item
            label="密码"
            prop="password"
            :rules="[
              { required: true, message: '请输入' },
              { min: 6, message: '密码不小于6位' }
            ]"
            v-if="!isEdit"
          >
            <el-input v-model="addObj.password" type="password" show-password :maxlength="20" />
          </el-form-item>

          <el-form-item
            label="确认密码"
            prop="password2"
            :rules="[
              { required: true, message: '请输入' },
              { validator: validatePassCheck, trigger: 'blur' }
            ]"
            v-if="!isEdit"
          >
            <el-input v-model="addObj.password2" type="password" show-password :maxlength="20" />
          </el-form-item>

          <el-form-item
            label="角色"
            prop="roleId"
            :rules="[{ required: true, message: '请选择角色' }]"
          >
            <el-select v-model="addObj.roleId">
              <el-option
                v-for="item in roleList"
                :label="item.name"
                :value="item.id"
                :key="item.id"
              />
            </el-select>
          </el-form-item>

          <el-form-item label="描述">
            <el-textarea
              :rows="5"
              placeholder="..."
              v-model="addObj.desc"
              :maxlength="200"
              show-word-limit
            />
          </el-form-item>
        </el-form>
      </div>
    </Dialog>
  </div>
</template>

<script lang="jsx">
import { editUser, addUser, getAllRole, getUser } from '../api.js'
import _ from 'lodash-es'
const defaultAddObj = {
  username: '',
  nickname: '',
  enableFlag: 'true',
  password: '',
  password2: '',
  roleId: null,
  desc: ''
}
export default {
  name: 'UserModal',
  components: {},
  props: {
    showTanent: {
      type: Boolean,
      default: false
    }
  },
  data() {
    return {
      loading: false,
      addObj: _.cloneDeep(defaultAddObj),
      title: '',

      isEdit: false,
      roleList: []
    }
  },
  created() {},
  methods: {
    validatePassCheck(rule, value, callback) {
      if (value === '') {
        callback(new Error('请再次输入密码'))
      } else if (value !== this.addObj.password) {
        callback(new Error('两次密码不一致!'))
      } else {
        callback()
      }
    },
    add() {
      this.addObj = _.cloneDeep(defaultAddObj)
      this.isEdit = false
      this.getAllRole()
      this.$refs.addModal.open({ title: '新增' })
    },
    edit(record) {
      this.isEdit = true
      this.loading = true
      getUser(record.id)
        .then((data1) => {
          if (data1.success) {
            const data = data1.result
            data.enableFlag = data.enableFlag + ''
            this.addObj = Object.assign({}, data)
            this.getAllRole()
            this.$refs.addModal.open({ title: '修改' })
          }
        })
        .finally(() => {
          this.loading = false
        })
    },
    close() {
      this.$refs.addModal.close()
    },
    handleOk() {
      // 触发表单验证
      this.$refs.addFormRef.validate((valid) => {
        // 验证表单没错误
        if (valid) {
          const values = _.cloneDeep(this.addObj)
          console.log('form values', values)
          values.enableFlag = values.enableFlag === 'true'
          this.loading = true
          let promise = null
          if (this.isEdit) {
            promise = editUser(values)
          } else {
            promise = addUser(values)
          }
          promise
            .then((data) => {
              if (data.success) {
                this.$message.success('操作成功')
                this.close()
                this.$emit('ok')
              } else {
                this.$message.success(data.message)
              }
            })
            .finally(() => {
              this.loading = false
            })
        }
      })
    },
    handleCancel() {
      this.$refs.addFormRef.clearValidate()
    },
    getAllRole() {
      getAllRole().then((resp) => {
        this.roleList = resp.result.list
      })
    }
  }
}
</script>

<style scoped></style>
