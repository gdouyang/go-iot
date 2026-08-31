<template>
  <div>
    <ContentWrap>
      <el-descriptions border :column="2">
        <template #title>
          产品信息
          <el-button link type="primary" @click="openBasicInfo">编辑</el-button>
        </template>
        <el-descriptions-item label="产品ID" :span="1">{{ data.id }}</el-descriptions-item>
        <el-descriptions-item label="网络类型" :span="1">{{
          data.networkType
        }}</el-descriptions-item>
        <el-descriptions-item label="时序保留" :span="1">{{ retentionLabel }}</el-descriptions-item>
        <el-descriptions-item label="时序存储" :span="1">{{
          storePolicyLabel
        }}</el-descriptions-item>
        <el-descriptions-item label="说明" :span="2">{{ data.desc }}</el-descriptions-item>
      </el-descriptions>

      <Network v-if="data.id" :product="data" />

      <Configuration :productId="data.id" :configuration="configuration" @refresh="refresh()" />
    </ContentWrap>

    <ProductAdd v-if="addVisible" ref="ProductAdd" @success="refresh()" />
  </div>
</template>

<script lang="jsx">
// import dayjs from 'dayjs'
// import _ from 'lodash-es'
import ProductAdd from '../modules/ProductAdd.vue'
import Configuration from './Configuration.vue'
import Network from './Network.vue'

export default {
  name: 'ProductInfo',
  components: {
    ProductAdd,
    Configuration,
    Network
  },
  props: {
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      configuration: [],
      addVisible: false
    }
  },
  computed: {
    retentionLabel() {
      const v = this.data?.retentionMonths
      if (v === null || v === undefined || v === '' || Number(v) <= 0) {
        return '跟随系统'
      }
      return `${v} 个月`
    },
    storePolicyLabel() {
      const p = this.data?.storePolicy || 'es'
      const map = {
        es: 'Elasticsearch',
        tdengine: 'TDengine',
        mock: 'Mock'
      }
      return map[p] || p
    }
  },
  watch: {
    'data.metaconfig'(newVal) {
      this.GetData()
    }
  },
  created() {
    this.GetData()
  },
  methods: {
    GetData() {
      const data = this.data
      this.configuration = data.metaconfig ? data.metaconfig : []
    },
    openBasicInfo() {
      this.addVisible = true
      this.$nextTick(() => {
        this.$refs.ProductAdd.edit(this.data)
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
