<template>
  <div>
    <el-descriptions border :column="2">
      <template #title>
        MQTT网络配置
        <el-button link type="primary" @click="openAdd">编辑</el-button>
        <NetworkRun :network="data" :productId="productId" @success="getData" />
      </template>
      <el-descriptions-item label="开启SSL" :span="1">{{
        data.configuration.useTLS ? '是' : '否'
      }}</el-descriptions-item>
      <el-descriptions-item label="连接地址" :span="1" v-if="data.type === 'MQTT_BROKER'">{{
        accessAddress
      }}</el-descriptions-item>
    </el-descriptions>
    <MqttConfigAdd ref="MqttConfigAdd" @success="getData" />
  </div>
</template>

<script lang="jsx">
// import dayjs from 'dayjs'
import MqttConfigAdd from './MqttConfigAdd.vue'
import { newMqttAddObj } from './entity.js'
import _ from 'lodash-es'
import Base from './Base.vue'

export default {
  name: 'MqttConfig',
  components: {
    MqttConfigAdd
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
      data: newMqttAddObj()
    }
  },
  computed: {},
  created() {
    if (!this.network) {
      this.getData()
    } else {
      const data = _.cloneDeep(this.network)
      this.convertConfiguration(data, newMqttAddObj())
      this.data = data
    }
  },
  methods: {
    getData() {
      this.getNetwork(this.productId, newMqttAddObj()).then((data) => {
        this.data = data
      })
    },
    openAdd() {
      this.$refs.MqttConfigAdd.open(this.productId)
    }
  }
}
</script>

<style lang="less" scoped></style>
