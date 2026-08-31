<template>
  <div>
    <Dialog ref="addModal" @confirm="handleOk" @close="handleCancel" :width="500">
      <el-form ref="addFormRef" :model="addObj" label-width="auto">
        <el-form-item
          label="角色名称"
          prop="name"
          :rules="[{ required: true, message: '请输入名称' }]"
        >
          <el-input placeholder="角色名称" :maxlength="20" v-model="addObj.name" show-word-limit />
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

        <el-divider>拥有权限</el-divider>
        <el-tree
          :data="treeData"
          show-checkbox
          node-key="id"
          :default-checked-keys="defaultCheckKeys"
          ref="treeObj"
        />
      </el-form>
    </Dialog>
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { getRole, editRole, addRole, getRoleRelMenus, getMenus } from '../api.js'
const defaultAddObj = {
  name: '',
  desc: ''
}
export default {
  name: 'RoleModal',
  components: {},
  data() {
    return {
      addObj: _.cloneDeep(defaultAddObj),
      spinning: 0,

      treeData: [],
      defaultCheckKeys: []
    }
  },
  created() {},
  methods: {
    add() {
      this.addObj = _.cloneDeep(defaultAddObj)
      this.isEdit = false
      this.getTreeData()
      this.$refs.addModal.open({ title: '新增' })
    },
    edit(record) {
      this.isEdit = true
      this.spinning++
      getRole(record.id)
        .then((data1) => {
          if (data1.success) {
            const data = data1.result
            this.$refs.addModal.open({ title: '修改' })
            this.addObj = data
            this.getTreeData()
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
      this.$refs.addFormRef.validate((valid) => {
        if (valid) {
          const values = _.cloneDeep(this.addObj)
          console.log('form values', values)
          values.ruleRefMenus = this.getCheckPermissions()
          let promise = null
          if (this.isEdit) {
            promise = editRole(values)
          } else {
            promise = addRole(values)
          }
          promise.then((data) => {
            if (data.success) {
              this.$message.success('操作成功')
              this.close()
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
    },
    getTreeData() {
      if (this.isEdit) {
        this.spinning++
        getRoleRelMenus(this.addObj.id)
          .then((data) => {
            if (data.success) {
              this.getAllMenu(data.result)
            }
          })
          .finally(() => {
            this.spinning--
          })
      } else {
        this.getAllMenu([])
      }
    },
    getAllMenu(checkeds) {
      this.spinning++
      this.defaultCheckKeys = []
      getMenus()
        .then((data) => {
          const datas = data.result
          const list = []
          let id = 1
          _.forEach(datas, (item) => {
            item.pid = 0
            item.id = id
            const checkItem = _.find(checkeds, (check) => check.code === item.permissionId)
            const parent = {
              id: id,
              pid: 0,
              label: item.permissionName,
              code: item.permissionId,
              checked: !_.isNil(checkItem)
            }
            list.push(parent)
            parent.children = []
            _.forEach(item.actionEntitySet, (ac) => {
              id++
              const actions = _.get(checkItem, 'action')
              const checked = !_.isNil(_.find(actions, (checkAc) => checkAc.id === ac.action))
              parent.children.push({
                id: id,
                pid: item.id,
                label: ac.describe,
                code: ac.action,
                checked: checked
              })
              if (checked) {
                this.defaultCheckKeys.push(id)
              }
            })
            id++
          })
          console.log(list)

          this.treeData = list
        })
        .finally(() => {
          this.spinning--
        })
    },
    getCheckPermissions() {
      var treeObj = this.$refs.treeObj
      var checkedNodes = treeObj.getCheckedNodes(false, true)
      const datas = []
      _.forEach(checkedNodes, (node) => {
        if (node.pid === 0) {
          const action = _.map(
            _.filter(checkedNodes, (n) => n.pid === node.id),
            (ac) => {
              return { id: ac.code, name: ac.name }
            }
          )
          datas.push({
            code: node.code,
            action: action
          })
        }
      })
      return datas
    }
  }
}
</script>

<style scoped></style>
