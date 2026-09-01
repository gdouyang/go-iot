<template>
  <ContentDetailWrap :header-border="false" v-loading="loading">
    <template #header>
      <el-row class="el-descriptions__title" style="align-items: center">
        <el-tooltip content="返回">
          <BaseButton @click="back" circle size="small"><Icon icon="el:ArrowLeft" /></BaseButton>
        </el-tooltip>
        <span class="detail-title">
          <span>产品：{{ detailData.name }}</span>
        </span>
        <el-tag :type="detailData.state ? 'success' : 'info'" round size="small">{{
          detailData.state ? '发布' : '停用'
        }}</el-tag>
        <span v-hasPermi="'product-mgr:save'">
          <span v-if="detailData.state">
            <el-popconfirm title="确认停用？" @confirm="unDeploy">
              <template #reference>
                <el-button link type="primary" class="link">停用</el-button>
              </template>
            </el-popconfirm>
          </span>
          <span v-if="!detailData.state">
            <el-popconfirm title="确认发布？" @confirm="deploy">
              <template #reference>
                <el-button link type="primary" class="link">发布</el-button>
              </template>
            </el-popconfirm>
          </span>
          <span v-if="detailData.state">
            <el-popconfirm title="确认重新应用该配置？" width="200px" @confirm="deploy">
              <template #reference>
                <span>
                  <el-tooltip content="修改物模型后需要重新应用配置" placement="top">
                    <el-button link type="primary" class="link">应用配置</el-button>
                  </el-tooltip>
                </span>
              </template>
            </el-popconfirm>
          </span>
        </span>
      </el-row>
    </template>
    <el-tabs model-value="info" class="product-detail-tabs">
      <el-tab-pane name="info" label="基本信息">
        <Info v-if="detailData.id" :data="detailData" @refresh="reloadProduct" />
      </el-tab-pane>
      <el-tab-pane name="tsl" label="物模型">
        <TSL
          v-if="detailData.id"
          :product="detailData"
          :propertyData="properties"
          :functionsData="functions"
          :eventsData="events"
          @refresh="reloadProduct"
          @save="updateData"
        />
      </el-tab-pane>
      <el-tab-pane name="codec" label="编解码">
        <Codec
          v-if="detailData.id"
          :id="GetId"
          :product="detailData"
          @refresh="reloadProduct"
          @save="updateData"
        />
      </el-tab-pane>
      <el-tab-pane v-if="detailData.networkType === 'MODBUS'" name="collector" label="点表/采集">
        <Collector
          v-if="detailData.id"
          :product="detailData"
          :properties="properties"
          @saved="reloadProduct"
        />
      </el-tab-pane>
    </el-tabs>
  </ContentDetailWrap>
</template>

<script lang="jsx">
import { deploy, undeploy, get, modifyTsl } from '@/views/iot/product/api.js'
import Info from './detail/Info.vue'
import TSL from './detail/TslIndex.vue'
import Codec from './detail/Codec.vue'
import Collector from './detail/Collector.vue'

export default {
  name: 'ProductDetialIndex',
  components: {
    Info,
    TSL,
    Codec,
    Collector
  },
  mixins: [],
  data() {
    return {
      loading: true,
      detailData: {},
      properties: [],
      functions: [],
      events: []
    }
  },
  computed: {
    GetId() {
      return this.$route.query.id
    }
  },
  mounted() {
    this.reloadProduct()
  },
  methods: {
    back() {
      this.$emit('back')
    },
    getDetail(id) {
      this.loading = true
      return get(id)
        .then((data) => {
          if (data.success) {
            return data.result
          }
        })
        .finally(() => {
          this.loading = false
        })
    },
    reloadProduct() {
      this.getDetail(this.GetId).then((result) => {
        if (result) {
          this.detailData = result
          if (result.metadata) {
            result.metadata = JSON.parse(result.metadata)
            const metadata = result.metadata
            this.events = metadata.events || []
            this.functions = metadata.functions || []
            this.properties = metadata.properties || []
          }
        }
      })
    },
    unDeploy() {
      const id = this.GetId
      undeploy(id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.reloadProduct()
        }
      })
    },
    deploy() {
      const id = this.GetId
      deploy(id).then((data) => {
        if (data.success) {
          this.$message.success('操作成功')
          this.reloadProduct()
        }
      })
    },
    updateData(type, item, onlySave) {
      const events = this.events
      const properties = this.properties
      const functions = this.functions
      let metadata = { events, properties, functions }
      if (type === 'event') {
        metadata = { events: item, properties, functions }
      } else if (type === 'properties') {
        metadata = { events, properties: item, functions }
      } else if (type === 'function') {
        metadata = { events, properties, functions: item }
      } else if (type === 'all') {
        metadata = { events: item.events, properties: item.properties, functions: item.functions }
      }
      const basicInfo = this.detailData
      const data = { metadata: JSON.stringify(metadata) }
      modifyTsl(basicInfo.id, data)
        .then((re) => {
          if (re.success) {
            this.$message.success('保存成功，如需生效请重新应用配置')
          }
        })
        .finally(() => {
          this.reloadProduct()
        })
    },
    toProductPage() {}
  }
}
</script>

<style lang="less" scoped>
.deviceInsTitle {
  display: flex;
  flex-direction: column;
}
.link {
  font-size: 12px;
  margin-left: 10px;
}
.product-detail-tabs {
  :deep(.el-tabs__content) {
    overflow: visible;
  }
}
</style>
