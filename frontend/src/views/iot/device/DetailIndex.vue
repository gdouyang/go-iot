<template>
  <ContentDetailWrap :header-border="false" v-loading="loading">
    <template #header>
      <el-row class="el-descriptions__title flex-item-center">
        <el-tooltip content="返回">
          <BaseButton @click="back" circle size="small"><Icon icon="el:ArrowLeft" /></BaseButton>
        </el-tooltip>
        <span class="detail-title">
          <span>设备：{{ getDeviceId }}</span>
        </span>
        <el-tag v-if="deviceState === 'online'" type="success" class="link">{{
          deviceStateText
        }}</el-tag>
        <el-tag v-else-if="deviceState === 'offline'" type="danger" class="link">{{
          deviceStateText
        }}</el-tag>
        <el-tag v-else type="info" class="link">{{ deviceStateText }}</el-tag>
        <el-tooltip v-if="deviceState === 'online'">
          <template #content>
            <div class="connection-info">
              <pre>{{ connectionInfo }}</pre>
            </div>
          </template>
          <el-button link type="success" class="link">连接信息</el-button>
        </el-tooltip>
        <DeviceCheckComponent :device-id="getDeviceId" />
        <span v-hasPermi="'device-mgr:save'">
          <el-popconfirm
            v-if="deviceState === 'online'"
            title="确认让此设备断开连接？"
            width="200px"
            @confirm="disconnectDevice"
          >
            <template #reference>
              <el-button link type="primary" class="link">断开连接</el-button>
            </template>
          </el-popconfirm>
          <el-popconfirm
            v-else-if="deviceState === 'noActive'"
            title="确认激活此设备？"
            width="200px"
            @confirm="changeDeploy"
          >
            <template #reference>
              <el-button link type="primary" class="link">激活设备</el-button>
            </template>
          </el-popconfirm>
          <el-popconfirm
            v-else-if="isNetClientType && deviceState === 'offline'"
            title="确认连接设备？"
            width="200px"
            @confirm="connectDevice"
          >
            <template #reference>
              <el-button link type="primary" class="link">连接</el-button>
            </template>
          </el-popconfirm>
          <el-button
            v-if="detailData.networkType === 'MODBUS' && deviceState === 'online'"
            link
            type="primary"
            class="link"
            @click="doCollectNow"
          >
            立即采集
          </el-button>
        </span>
        <el-tooltip content="刷新">
          <BaseButton @click="reloadDevice" circle size="small" class="link"
            ><Icon icon="el:Refresh"
          /></BaseButton>
        </el-tooltip>
      </el-row>
    </template>
    <div>
      <el-descriptions :column="3">
        <el-descriptions-item label="ID">{{ detailData.id }}</el-descriptions-item>
        <el-descriptions-item label="名称">{{ detailData.name }}</el-descriptions-item>
        <el-descriptions-item label="产品">
          {{ detailData.productName }}
        </el-descriptions-item>
      </el-descriptions>
    </div>
    <el-tabs v-model="activeTabKey">
      <el-tab-pane name="info" label="基本信息" />
      <el-tab-pane name="status" label="运行状态" />
      <el-tab-pane name="properties" label="设备属性" />
      <el-tab-pane name="function" label="设备功能" />
      <el-tab-pane name="events" label="设备事件" />
      <el-tab-pane name="log" label="日志" />
    </el-tabs>
    <template v-if="activeTabKey === 'info'">
      <Info v-if="detailData.id" :device="detailData" @refresh="reloadDevice" />
    </template>
    <template v-if="activeTabKey === 'status'">
      <Status
        v-if="detailData.id"
        :device="detailData"
        @refresh="reloadDevice"
        :realtimeData="realtimeData"
      />
    </template>
    <template v-if="activeTabKey === 'function'">
      <Function v-if="detailData.id" :device="detailData" />
    </template>
    <template v-if="activeTabKey === 'log'">
      <Log v-if="detailData.id" :deviceId="detailData.id" />
    </template>
    <template v-if="activeTabKey === 'properties'">
      <Properties v-if="detailData.id" :device="detailData" />
    </template>
    <template v-if="activeTabKey === 'events'">
      <Events v-if="detailData.id" :device="detailData" />
    </template>
  </ContentDetailWrap>
</template>

<script lang="jsx">
import {
  getStatusText,
  getDetail,
  getConnectionInfo,
  connect,
  disconnect,
  deploy,
  collectNow,
  getEventBusUrl
} from './api.js'
import DeviceCheckComponent from './detail/DeviceCheckComponent.vue'
import { EventBusWs } from '@/utils/eventBusWs'
import Info from './detail/Info.vue'
import Status from './detail/Status.vue'
import Function from './detail/Function.vue'
import Log from './detail/Log.vue'
import Properties from './detail/Properties.vue'
import Events from './detail/Events.vue'

export default {
  name: 'DeviceDetail',
  components: {
    DeviceCheckComponent,
    Info,
    Status,
    Function,
    Log,
    Properties,
    Events
  },
  data() {
    return {
      loading: true,
      detailData: {},
      activeTabKey: 'info',
      tableList: [
        { key: 'info', tab: '实例信息' },
        { key: 'status', tab: '运行状态' },
        { key: 'function', tab: '设备功能' },
        { key: 'properties', tab: '设备属性' },
        { key: 'events', tab: '设备事件' },
        { key: 'log', tab: '日志' }
      ],
      realtimeData: {},
      connectionInfo: {},
      eventWs: null
    }
  },
  computed: {
    getDeviceId() {
      return this.$route.query.id
    },
    deviceState() {
      const status = this.detailData.state
      return status
    },
    deviceStateText() {
      const status = this.detailData.state
      return getStatusText(status)
    },
    isNetClientType() {
      return (
        this.detailData.networkType === 'TCP_CLIENT' ||
        this.detailData.networkType === 'MQTT_CLIENT' ||
        this.detailData.networkType === 'MODBUS'
      )
    }
  },
  created() {
    this.eventWs = new EventBusWs(getEventBusUrl(this.getDeviceId, '*'), (evt) =>
      this.onWsMessage(evt)
    )
    this.eventWs.connect()
  },
  mounted() {
    const { id } = this.$route.query
    this.getDeviceDetail(id).then((result) => {
      if (result) {
        this.detailData = result
        if (result.state === 'online') {
          this.getConnectionInfo()
        }
      }
    })
  },
  unmounted() {
    if (this.eventWs) {
      this.eventWs.close()
    }
  },
  methods: {
    back() {
      this.$emit('back')
    },
    onTabChange(key) {
      if (!this.detailData.metadata) {
        this.$message.error('请检查产品物模型')
        return
      }
      this.activeTabKey = key
    },
    getDeviceDetail(id) {
      this.loading = true
      return getDetail(id)
        .then((data) => {
          if (data.success) {
            return data.result
          }
        })
        .finally(() => {
          this.loading = false
        })
    },
    reloadDevice() {
      this.getDeviceDetail(this.getDeviceId).then((result) => {
        if (result) {
          this.detailData = result
          this.getConnectionInfo()
        }
      })
    },
    changeDeploy() {
      const deviceId = this.getDeviceId
      deploy(deviceId).then((data) => {
        if (data.success) {
          this.$message.success('激活成功')
          this.reloadDevice()
        }
      })
    },
    getConnectionInfo() {
      getConnectionInfo(this.getDeviceId).then((data) => {
        if (data.success) {
          this.connectionInfo = JSON.stringify(data.result, null, 2)
        }
      })
    },
    disconnectDevice() {
      const deviceId = this.getDeviceId
      disconnect(deviceId).then((data) => {
        if (data.success) {
          this.$message.success('断开连接成功')
          this.reloadDevice()
        }
      })
    },
    doCollectNow() {
      collectNow(this.getDeviceId).then((data) => {
        if (data.success) {
          this.$message.success('已触发采集')
        }
      })
    },
    connectDevice() {
      const deviceId = this.getDeviceId
      connect(deviceId).then((data) => {
        if (data.success) {
          this.$message.success('连接成功')
          this.reloadDevice()
        }
      })
    },
    onWsMessage(evt) {
      var data = JSON.parse(evt.data)
      if (data.type === 'online') {
        this.detailData.state = 'online'
        this.getConnectionInfo()
      } else if (data.type === 'offline') {
        this.detailData.state = 'offline'
      } else if (data.type === 'property' || data.type === 'event') {
        this.realtimeData = data
      }
    }
  }
}
</script>

<style lang="less" scoped>
.deviceInsTitle {
  display: flex;
  flex-direction: column;
}
.connection-info {
  max-height: 500px;
  max-width: 400px;
  overflow: auto;
  pre {
    white-space: break-spaces;
  }
}
.link {
  margin-right: 5px;
}
</style>
