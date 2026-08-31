<template>
  <el-drawer
    title="选择品类"
    width="500"
    visible
    :afterVisibleChange="visibleChange"
    @close="visibleChange(false)"
  >
    <div style="padding-bottom: 20px">
      <el-select style="width: 43%" defaultValue="all" @change="findCategoryByPid">
        <el-option value="all">全部领域</el-option>
        <el-option v-for="item in categoryLIst" :key="item.id" :value="item.id">
          {item.name}
        </el-option>
      </el-select>
      <el-input-search
        allowClear
        placeholder="请输入品类名称或者所属场景"
        enterButton
        @change="
          (event) => {
            if (event.target.value === '') {
              assemblyData()
            }
          }
        "
        @search="findCategoryByVague"
        :style="{ width: '43%', paddingLeft: 20 }"
      />
    </div>
    <div>
      <el-table
        :dataSource="categoryAllLIst || []"
        :columns="columns"
        rowKey="id"
        :pagination="{ pageSize: 8 }"
      >
        <span slot="action" slot-scope="text, record">
          <span v-if="data.id === record.id">已选择</span>
          <a v-else @click="choice(record)">选择</a>
        </span>
      </el-table>
    </div>
    <div class="drawer-footer">
      <el-button style="margin-right: 8px" @click="visibleChange(false)">关闭</el-button>
    </div>
  </el-drawer>
</template>

<script lang="jsx">
import _ from 'lodash-es'
export default {
  name: 'Classified',
  props: {
    visible: {
      type: Boolean,
      default: false
    },
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      openFlag: false,
      categoryLIst: [],
      categoryAllLIst: [],
      columns: [
        {
          label: '品类名称',
          field: 'name'
        },
        {
          label: '所属场景',
          field: 'parentName'
        },
        {
          label: '操作',
          width: '120px',
          align: 'center',
          scopedSlots: { customRender: 'action' }
        }
      ]
    }
  },
  watch: {
    visible(newVal) {
      this.openFlag = newVal
    }
  },
  created() {
    this.deviceCategoryTree()
  },
  methods: {
    assemblyData() {
      const list = []
      this.categoryLIst.map((item) => {
        _.map(item.children, (i) => {
          if (i.children) {
            i.children.map((data) => {
              data['parentName'] = i.name
              data['categoryId'] = [item.id, i.id, data.id]
              list.push(data)
            })
          } else {
            i['parentName'] = item.name
            i['categoryId'] = [item, i.id]
            list.push(i)
          }
        })
      })
      this.categoryAllLIst = list
    },
    findCategoryByPid(value) {
      this.categoryAllLIst.splice(0, this.categoryAllLIst.length)
      if (value === 'all') {
        this.assemblyData()
      } else {
        const category = this.categoryLIst.find((i) => i.id === value) || {}
        const list = []
        _.map(category.children, (i) => {
          if (i.children) {
            i.children.map((data) => {
              data['parentName'] = i.name
              data['categoryId'] = [value, i.id, data.id]
              list.push(data)
            })
          } else {
            i['parentName'] = category.name
            i['categoryId'] = [value, i.id]
            list.push(i)
          }
          this.categoryAllLIst = list
        })
      }
    },
    findCategoryByVague(value) {
      if (value === '') {
        this.categoryAllLIst.splice(0, this.categoryAllLIst.length)
        this.assemblyData()
      } else {
        const list = []
        this.categoryAllLIst.map((item) => {
          if (item.name.indexOf(value) !== -1 || item.parentName.indexOf(value) !== -1) {
            list.push(item)
          }
        })
        this.categoryAllLIst.splice(0, this.categoryAllLIst.length)
        this.categoryAllLIst = list
      }
    },
    deviceCategoryTree() {
      this.$http.get(`/device/category/_tree`).then((data) => {
        data.result.map((item) => {
          _.map(item.children, (i) => {
            if (i.children) {
              i.children.map((data) => {
                data['parentName'] = i.name
                data['categoryId'] = [item.id, i.id, data.id]
                this.categoryAllLIst.push(data)
              })
            } else {
              i['parentName'] = item.name
              i['categoryId'] = [item.id, i.id]
              this.categoryAllLIst.push(i)
            }
          })
        })
        this.categoryLIst = data.result
      })
    },
    visibleChange(flag) {
      if (!flag) {
        this.$emit('close')
      }
    },
    choice(data) {
      this.$emit('choice', data)
    }
  }
}
</script>

<style lang="less" scoped></style>
