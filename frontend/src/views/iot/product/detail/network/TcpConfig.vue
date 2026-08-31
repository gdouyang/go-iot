<template>
  <div>
    <el-descriptions border :column="2">
      <template #title>
        TCP网络配置
        <el-button link type="primary" @click="openAdd">编辑</el-button>
        <NetworkRun :network="data" :productId="productId" @success="getData" />
      </template>
      <el-descriptions-item label="开启SSL" :span="1">{{
        data.configuration.useTLS ? '是' : '否'
      }}</el-descriptions-item>
      <el-descriptions-item label="解析方式" :span="1">{{
        parserType(data.configuration.delimeter.type)
      }}</el-descriptions-item>
      <el-descriptions-item label="连接地址" :span="2" v-if="data.type === 'TCP_SERVER'">{{
        accessAddress
      }}</el-descriptions-item>
    </el-descriptions>
    <TcpConfigAdd ref="TcpConfigAdd" @success="getData" />
  </div>
</template>

<script lang="jsx">
// import dayjs from 'dayjs'
import TcpConfigAdd from './TcpConfigAdd.vue'
import { newTcpAddObj, parserType } from './entity.js'
import _ from 'lodash-es'
import Base from './Base.vue'

export default {
  name: 'TcpConfig',
  components: {
    TcpConfigAdd
  },
  mixins: [Base],
  props: {
    productId: {
      type: String,
      default: null
    },
    network: {
      type: Object,
      default: () => null
    }
  },
  data() {
    return {
      data: newTcpAddObj()
    }
  },
  computed: {},
  created() {
    if (!this.network) {
      this.getData()
    } else {
      const data = _.cloneDeep(this.network)
      this.convertConfiguration(data, newTcpAddObj())
      this.data = data
    }
  },
  methods: {
    parserType(type) {
      return parserType(type)
    },
    getData() {
      this.getNetwork(this.productId, newTcpAddObj()).then((data) => {
        this.data = data
      })
    },
    openAdd() {
      this.$refs.TcpConfigAdd.open(this.productId)
    }
  }
}
</script>

<style lang="less" scoped></style>
