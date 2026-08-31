<template>
  <div>
    <ContentWrap>
      <el-descriptions border :column="2">
        <template #title>
          <div class="flex-item-center">
            设备信息
            <el-button link type="primary" v-hasPermi="'device-mgr:save'" @click="openBasicInfo"
              >编辑</el-button
            >
          </div>
        </template>
        <el-descriptions-item label="产品名称" :span="1">{{
          device.productName
        }}</el-descriptions-item>
        <el-descriptions-item label="创建时间" :span="1">{{ GetCreateTime }}</el-descriptions-item>
        <el-descriptions-item label="说明" :span="2">{{ device.desc }}</el-descriptions-item>
      </el-descriptions>

      <Configuration :device="device" @refresh="refresh" />
    </ContentWrap>

    <DeviceAdd v-if="deviceVisible" ref="DeviceAdd" @success="refresh" />
  </div>
</template>

<script lang="jsx">
import dayjs from 'dayjs'
import { updateLocation } from '@/views/iot/device/api.js'
import DeviceAdd from '../modules/DeviceAdd.vue'
import Configuration from './Configuration.vue'

export default {
  name: 'DeviceInfo',
  components: {
    DeviceAdd,
    Configuration
  },
  props: {
    device: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      deviceVisible: false,
      extendData: {
        deviceId: null,
        longitude: null, // 经度
        latitude: null // 纬度
      }
    }
  },
  computed: {
    GetCreateTime() {
      return dayjs(this.device.createTime).format('YYYY-MM-DD HH:mm:ss')
    },
    deviceId() {
      const { id } = this.device
      return id
    }
  },
  created() {},
  methods: {
    refresh() {
      this.$emit('refresh')
    },
    openBasicInfo() {
      this.deviceVisible = true
      this.$nextTick(() => {
        this.$refs.DeviceAdd.edit(this.device)
      })
    },
    editLocation() {
      this.$refs.LocationConfig.open(this.extendData)
    },
    selectLocation(value) {
      const param = {
        deviceId: this.extendData.deviceId,
        longitude: value.longitude,
        latitude: value.latitude
      }
      updateLocation(param).then((resp) => {
        if (resp.success) {
          this.$message.success('操作成功')
          this.$refs.LocationConfig.close()
        } else {
          this.$message.success(resp.message)
        }
      })
    }
  }
}
</script>

<style lang="less" scoped></style>
