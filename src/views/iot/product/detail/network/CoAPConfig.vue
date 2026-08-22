<template>
  <div>
    <el-descriptions border :column="2">
      <template #title>
        CoAP网络配置
        <el-button type="primary" link @click="openAdd">编辑</el-button>
        <NetworkRun :network="data" :productId="productId" @success="getData" />
      </template>
      <!-- <el-descriptions-item label="开启SSL" :span="1">{{
        data.configuration.useTLS ? '是' : '否'
      }}</el-descriptions-item> -->
      <el-descriptions-item label="连接地址" :span="1">
        <div v-for="a in accessAddress" :key="a">{{ a }}</div>
      </el-descriptions-item>
    </el-descriptions>
    <CoAPConfigAdd ref="CoAPConfigAdd" @success="getData" />
  </div>
</template>

<script lang="jsx">
// import dayjs from 'dayjs'
import CoAPConfigAdd from './CoAPConfigAdd.vue'
import { newHttpAddObj } from './entity.js'
import _ from 'lodash-es'
import Base from './Base.vue'

export default {
  name: 'CoAPConfig',
  components: {
    CoAPConfigAdd
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
      data: newHttpAddObj()
    }
  },
  computed: {
    accessAddress() {
      const port = _.get(this.data, 'port', '')
      if (!port) {
        return ''
      }
      const ssl = _.get(this.data, 'configuration.useTLS', false)
      const address = (ssl ? 'https://' : 'http://') + this.accessIp + ':' + port
      const routers = _.get(this.data, 'configuration.routers', [])
      if (!_.isEmpty(routers)) {
        const arr = []
        _.forEach(routers, (r) => {
          arr.push(address + r.url)
        })
        return arr
      }
      return [address]
    }
  },
  created() {
    if (!this.network) {
      this.getData()
    } else {
      const data = _.cloneDeep(this.network)
      this.convertConfiguration(data, newHttpAddObj())
      this.data = data
    }
  },
  methods: {
    getData() {
      this.getNetwork(this.productId, newHttpAddObj()).then((data) => {
        this.data = data
      })
    },
    openAdd() {
      this.$refs.CoAPConfigAdd.open(this.productId)
    }
  }
}
</script>

<style lang="less" scoped></style>
