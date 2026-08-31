<template>
  <div style="width: 100%; margin-top: 10px">
    <el-descriptions>
      <template #title>
        配置
        <el-button type="primary" link @click="addConfig">添加</el-button>
      </template>
    </el-descriptions>

    <el-descriptions border :column="2">
      <el-descriptions-item v-for="(item, index) in configuration" :key="index">
        <template #label>
          <el-tooltip :content="item.desc">
            {{ item.property }}
          </el-tooltip>
          <BaseButton class="prop-edit" @click="modifyConfig(item)" circle size="small" title="编辑"
            ><Icon icon="el:Edit"
          /></BaseButton>
          <el-popconfirm
            v-if="!item.buildin"
            title="确认删除配置？"
            width="200px"
            @confirm="deleteConfig(item)"
          >
            <template #reference>
              <BaseButton class="prop-edit" circle size="small"
                ><Icon icon="el:Delete"
              /></BaseButton>
            </template>
          </el-popconfirm>
        </template>
        <span v-if="item.type == 'password' && item.value">••••••</span>
        <span v-else>{{ item.value }}</span>
      </el-descriptions-item>
    </el-descriptions>

    <ConfigurationAdd
      v-if="updateVisibleAdd"
      :productId="productId"
      :all-config="configuration"
      :data="currentConfig"
      @close="
        () => {
          updateVisibleAdd = false
          refresh()
        }
      "
    />
  </div>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { updateProduct } from '@/views/iot/product/api.js'
import ConfigurationAdd from './ConfigurationAdd.vue'
export default {
  name: 'Configuration',
  components: {
    ConfigurationAdd
  },
  props: {
    productId: {
      type: String,
      default: () => null
    },
    configuration: {
      type: Array,
      default: () => []
    }
  },
  data() {
    return {
      updateVisibleAdd: false,
      currentConfig: null
    }
  },
  methods: {
    refresh() {
      this.$emit('refresh')
    },
    addConfig() {
      this.currentConfig = null
      this.updateVisibleAdd = true
    },
    modifyConfig(data) {
      this.currentConfig = data
      this.updateVisibleAdd = true
    },
    deleteConfig(data) {
      const conf = _.filter(this.configuration, (p) => p.property !== data.property)
      const param = {
        id: this.productId,
        metaconfig: conf
      }
      updateProduct(this.productId, param).then((resp) => {
        if (resp.success) {
          this.$message.success('操作成功')
          this.refresh()
        }
      })
    }
  }
}
</script>

<style lang="less" scoped>
.link {
  margin-left: 10px;
}
.prop-edit {
  margin-left: 5px;
}
</style>
