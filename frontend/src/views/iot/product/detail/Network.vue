<template>
  <div style="margin-top: 10px">
    <TcpConfig
      v-if="network.type === 'TCP_SERVER' || network.type === 'TCP_CLIENT'"
      :productId="product.id"
      :network="network"
    />
    <MqttConfig
      v-if="network.type === 'MQTT_BROKER' || network.type === 'MQTT_CLIENT'"
      :productId="product.id"
      :network="network"
    />
    <WebSocketConfig
      v-if="network.type === 'WEBSOCKET_SERVER'"
      :productId="product.id"
      :network="network"
    />
    <HttpConfig v-if="network.type === 'HTTP_SERVER'" :productId="product.id" :network="network" />
    <CoAPConfig v-if="network.type === 'COAP_SERVER'" :productId="product.id" :network="network" />
  </div>
</template>

<script lang="jsx">
// import dayjs from 'dayjs'
// import _ from 'lodash-es'
import { getNetwork } from '@/views/iot/product/api.js'
import TcpConfig from './network/TcpConfig.vue'
import MqttConfig from './network/MqttConfig.vue'
import WebSocketConfig from './network/WebSocketConfig.vue'
import HttpConfig from './network/HttpConfig.vue'
import CoAPConfig from './network/CoAPConfig.vue'

export default {
  name: 'Network',
  components: {
    TcpConfig,
    MqttConfig,
    WebSocketConfig,
    HttpConfig,
    CoAPConfig
  },
  props: {
    product: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      network: {}
    }
  },
  computed: {},
  created() {
    this.GetData()
  },
  methods: {
    GetData() {
      const { id } = this.product
      getNetwork(id).then((data) => {
        if (data.success) {
          this.network = data.result || {}
        }
      })
    },
    refresh() {
      this.$emit('refresh')
      this.GetData()
    }
  }
}
</script>

<style lang="less" scoped></style>
