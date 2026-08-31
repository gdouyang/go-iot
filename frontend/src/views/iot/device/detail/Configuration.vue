<template>
  <div style="width: 100%; margin-top: 10px" v-if="configuration.length">
    <el-descriptions :column="3" border>
      <template #title>
        <div class="flex-item-center">
          配置
          <span v-hasPermi="'device-mgr:save'">
            <el-popconfirm title="确认重新应用该配置？" width="200px" @confirm="changeDeploy">
              <template #reference>
                <el-button link type="primary">应用配置</el-button>
              </template>
            </el-popconfirm>
          </span>
          <el-tooltip content="修改配置后需重新应用后才能生效。">
            <Icon icon="el:QuestionFilled" />
          </el-tooltip>
          <span v-if="canResetConfig" v-hasPermi="'device-mgr:save'">
            <el-popconfirm title="确认恢复默认配置？" width="200px" @confirm="configurationReset">
              <template #reference>
                <el-button link type="primary">恢复默认</el-button>
              </template>
            </el-popconfirm>
            <el-tooltip
              :title="`该设备单独编辑过[${deviceConfigKeys}]，点击此将恢复成默认的配置信息，请谨慎操作。`"
            >
              <Icon icon="el:QuestionFilled" />
            </el-tooltip>
          </span>
        </div>
      </template>
    </el-descriptions>

    <el-descriptions border :column="2" title="">
      <el-descriptions-item v-for="(item, inx) in configuration" :key="inx">
        <template #label>
          <el-tooltip :content="item.desc">
            <span>{{ item.property }}</span>
          </el-tooltip>
          <BaseButton
            class="prop-edit"
            v-hasPermi="'device-mgr:save'"
            circle
            size="small"
            title="编辑"
            @click="editConfigItem(item)"
            ><Icon icon="el:Edit"
          /></BaseButton>
        </template>
        <span>{{ getValue(item) }}</span>
      </el-descriptions-item>
    </el-descriptions>

    <ConfigurationUpdate
      v-if="updateVisible"
      :deviceId="device.id"
      :data="configItem"
      :all-config="deviceConfig"
      @close="
        () => {
          updateVisible = false
        }
      "
      @refresh="refresh"
    />
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { deploy, updateDevice } from '@/views/iot/device/api.js'
import { getMetaconfig } from '@/views/iot/product/api.js'
import ConfigurationUpdate from './ConfigurationUpdate.vue'
export default {
  name: 'Configuration',
  components: {
    ConfigurationUpdate
  },
  props: {
    device: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      configuration: [],
      updateVisible: false,
      configItem: {}
    }
  },
  computed: {
    deviceConfig() {
      return this.device.metaconfig ? this.device.metaconfig : {}
    },
    canResetConfig() {
      return !_.isEmpty(this.deviceConfig)
    },
    deviceConfigKeys() {
      return _.keys(this.deviceConfig)
    },
    deviceId() {
      return this.device.id
    }
  },
  created() {
    this.GetData()
  },
  methods: {
    getValue(item) {
      const has = _.has(this.deviceConfig, item.property)
      let value = item.value
      if (has) {
        value = _.get(this.deviceConfig, item.property)
      }
      if (item.type === 'password' && !_.isEmpty(value)) {
        return '••••••'
      }
      return value
    },
    GetData() {
      const id = this.device.productId
      getMetaconfig(id).then((data) => {
        this.configuration = data
      })
    },
    changeDeploy() {
      deploy(this.deviceId).then((data) => {
        if (data.success) {
          this.$message.success('应用成功')
          this.refresh()
        }
      })
    },
    refresh() {
      this.$emit('refresh')
    },
    editConfigItem(item) {
      this.configItem = _.cloneDeep(item)
      const has = _.has(this.deviceConfig, item.property)
      if (has) {
        this.configItem.value = _.get(this.deviceConfig, item.property)
      }
      this.updateVisible = true
    },
    configurationReset() {
      updateDevice({
        id: this.deviceId,
        metaconfig: {}
      }).then((resp) => {
        if (resp.success) {
          this.$message.success('恢复默认配置成功')
          this.refresh()
        }
      })
    }
  }
}
</script>

<style lang="less" scoped>
.prop-edit {
  margin-left: 5px;
}
</style>
