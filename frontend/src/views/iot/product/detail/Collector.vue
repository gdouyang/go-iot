<template>
  <div class="collector-page">
    <el-alert
      type="info"
      :closable="false"
      show-icon
      title="采集由组+点表完成（组包读寄存器）。编解码可实现 OnMessage 做二次处理：GetMessage() 为 {source:'collect', groupId, properties}；有 OnMessage 时由脚本 SaveProperties，没有则采集器直接入库。写命令仍走 OnInvoke。"
      style="margin-bottom: 12px"
    />
    <el-form label-width="140px" class="mb-12">
      <el-form-item label="启用采集">
        <el-switch v-model="form.enabled" />
      </el-form-item>
      <el-form-item label="地址基准">
        <el-radio-group v-model="form.addressBase">
          <el-radio :value="0">
            <span>PDU 0 起始</span>
            <el-tooltip placement="top" :show-after="200">
              <template #content>
                点表填多少，报文里就是多少，不做加减。<br />
                适合说明书写「地址 0」的情况。<br />
                例：填 4，发出去的起始地址是 4。
              </template>
              <Icon icon="el:QuestionFilled" class="addr-tip-icon" @click.stop />
            </el-tooltip>
          </el-radio>
          <el-radio :value="1">
            <span>协议 1 起始</span>
            <el-tooltip placement="top" :show-after="200">
              <template #content>
                按说明书从 1 数的编号填写，发送时自动减 1。<br />
                适合第一个寄存器写成 1（或 40001 对应偏移 1）的手册。<br />
                例：填 1，实际读取报文地址 0。
              </template>
              <Icon icon="el:QuestionFilled" class="addr-tip-icon" @click.stop />
            </el-tooltip>
          </el-radio>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="连续失败下线">
        <el-input-number
          v-model="form.offlineAfterFailures"
          :min="0"
          :max="100"
          controls-position="right"
        />
        <span class="hint">0 表示读失败仍保持在线</span>
      </el-form-item>
    </el-form>

    <h4>采集组</h4>
    <el-table :data="form.groups" border size="small" class="mb-12 collector-table">
      <el-table-column label="标识" width="160">
        <template #default="{ row }">
          <el-input v-model="row.id" maxlength="32"/>
        </template>
      </el-table-column>
      <el-table-column label="名称" min-width="160">
        <template #default="{ row }">
          <el-input v-model="row.name" maxlength="64"/>
        </template>
      </el-table-column>
      <el-table-column label="间隔(ms)" width="168">
        <template #default="{ row }">
          <el-input-number v-model="row.intervalMs" :min="100" :step="100" controls-position="right" />
        </template>
      </el-table-column>
      <el-table-column label="上报" width="160">
        <template #default="{ row }">
          <el-select v-model="row.report">
            <el-option value="onPeriod" label="每周期" />
            <el-option value="onChange" label="变化" />
            <el-option value="onChangeOrPeriod" label="变化或心跳" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="72" align="center">
        <template #default="{ $index }">
          <el-button link type="danger" @click="form.groups.splice($index, 1)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button class="mb-16" @click="addGroup">添加组</el-button>

    <h4>点表</h4>
    <el-table :data="form.points" border size="small" class="collector-table">
      <el-table-column label="点 ID" width="140">
        <template #default="{ row }">
          <el-input v-model="row.id" maxlength="32" />
        </template>
      </el-table-column>
      <el-table-column label="物模型属性" min-width="180">
        <template #default="{ row }">
          <el-select v-model="row.propertyId" filterable>
            <el-option
              v-for="p in propertyList()"
              :key="p.id"
              :label="`${p.name} (${p.id})`"
              :value="p.id"
            />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="组" width="100">
        <template #default="{ row }">
          <el-select v-model="row.groupId">
            <el-option v-for="g in form.groups" :key="g.id" :label="g.id" :value="g.id" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column label="表" width="140">
        <template #default="{ row }">
          <el-select v-model="row.table">
            <el-option value="HOLDING_REGISTERS" label="保持寄存器" />
            <el-option value="INPUT_REGISTERS" label="输入寄存器" />
            <el-option value="COILS" label="线圈" />
            <el-option value="DISCRETES_INPUT" label="离散输入" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column width="148">
        <template #header>
          <span class="col-head">
            地址
            <el-tooltip placement="top" :show-after="200">
              <template #content>
                寄存器/线圈地址，含义由上方「地址基准」决定。<br />
                PDU 0 起始：填多少，报文里就是多少。<br />
                协议 1 起始：按手册从 1 数，发送时自动减 1。
              </template>
              <Icon icon="el:QuestionFilled" class="addr-tip-icon" @click.stop />
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">
          <el-input-number v-model="row.address" :min="0" :max="65535" controls-position="right" />
        </template>
      </el-table-column>
      <el-table-column label="类型" width="112">
        <template #default="{ row }">
          <el-select v-model="row.dataType">
            <el-option v-for="t in dataTypes" :key="t" :label="t" :value="t" />
          </el-select>
        </template>
      </el-table-column>
      <el-table-column width="128">
        <template #header>
          <span class="col-head">
            缩放
            <el-tooltip placement="top" :show-after="200">
              <template #content>
                入库值 = 原始值 × 缩放。<br />
                例：寄存器 105，缩放 0.1，得到 10.5。<br />
                不填或 0 按 1 处理。
              </template>
              <Icon icon="el:QuestionFilled" class="addr-tip-icon" @click.stop />
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">
          <el-input-number v-model="row.scale" :step="0.1" controls-position="right" />
        </template>
      </el-table-column>
      <el-table-column width="108">
        <template #header>
          <span class="col-head">
            字节序
            <el-tooltip placement="top" :show-after="200">
              <template #content>
                16 位：AB（高字节在前）或 BA。<br />
                32/64 位：ABCD、CDAB、BADC、DCBA。<br />
                空则 16 位按 AB，32/64 位按 ABCD。
              </template>
              <Icon icon="el:QuestionFilled" class="addr-tip-icon" @click.stop />
            </el-tooltip>
          </span>
        </template>
        <template #default="{ row }">
          <el-input v-model="row.byteOrder" placeholder="AB" />
        </template>
      </el-table-column>
      <el-table-column label="操作" width="72" align="center">
        <template #default="{ $index }">
          <el-button link type="danger" @click="form.points.splice($index, 1)">删除</el-button>
        </template>
      </el-table-column>
    </el-table>
    <el-button class="mt-8" @click="addPoint">添加点</el-button>

    <div class="collector-actions">
      <el-button type="primary" :loading="saving" @click="save">保存采集配置</el-button>
    </div>
  </div>
</template>

<script>
import { getCollector, saveCollector } from '@/views/iot/product/api.js'

const emptyForm = () => ({
  version: 1,
  enabled: true,
  addressBase: 0,
  offlineAfterFailures: 0,
  groups: [],
  points: []
})

export default {
  name: 'ProductCollector',
  props: {
    product: { type: Object, default: () => ({}) },
    properties: { type: Array, default: () => [] }
  },
  data() {
    return {
      saving: false,
      form: emptyForm(),
      dataTypes: [
        'int16',
        'uint16',
        'int32',
        'uint32',
        'float32',
        'int64',
        'uint64',
        'float64',
        'bool',
        'string'
      ]
    }
  },
  watch: {
    'product.id': {
      immediate: true,
      handler() {
        this.load()
      }
    }
  },
  methods: {
    propertyList() {
      return Array.isArray(this.properties) ? this.properties : []
    },
    parsedCollector(raw) {
      const base = emptyForm()
      let src = raw
      if (!src) return base
      if (typeof src === 'string') {
        try {
          src = JSON.parse(src)
        } catch {
          return base
        }
      }
      const merged = { ...base, ...src }
      merged.groups = Array.isArray(merged.groups) ? merged.groups.slice() : []
      merged.points = Array.isArray(merged.points) ? merged.points.slice() : []
      return merged
    },
    ensureLists() {
      if (!Array.isArray(this.form.groups)) {
        this.form.groups = []
      }
      if (!Array.isArray(this.form.points)) {
        this.form.points = []
      }
    },
    load() {
      if (!this.product?.id) return
      getCollector(this.product.id)
        .then((resp) => {
          if (resp && resp.success) {
            this.form = this.parsedCollector(resp.result)
          }
        })
        .catch(() => {})
    },
    addGroup() {
      this.ensureLists()
      this.form.groups.push({
        id: 'g' + (this.form.groups.length + 1),
        name: '',
        intervalMs: 1000,
        report: 'onPeriod',
        maxQuantity: 125,
        gapTolerance: 0
      })
    },
    addPoint() {
      this.ensureLists()
      if (!this.form.groups.length) {
        this.addGroup()
      }
      const props = this.propertyList()
      this.form.points.push({
        id: 'p' + (this.form.points.length + 1),
        propertyId: props[0]?.id || '',
        groupId: this.form.groups[0]?.id || '',
        table: 'HOLDING_REGISTERS',
        address: 0,
        dataType: 'int16',
        scale: 1,
        byteOrder: 'AB',
        access: 'read'
      })
    },
    save() {
      this.ensureLists()
      if (this.form.points.length && !this.form.groups.length) {
        this.$message.warning('已配置点表，请先添加采集组')
        return
      }
      const groupIds = new Set(this.form.groups.map((g) => g.id))
      if (this.form.points.some((p) => !p.groupId || !groupIds.has(p.groupId))) {
        this.$message.warning('点表中有点未关联有效采集组')
        return
      }
      if (this.form.enabled && !this.form.points.length) {
        this.$message.warning('启用采集时请至少配置一个点')
        return
      }
      this.saving = true
      saveCollector(this.product.id, this.form)
        .then((resp) => {
          if (resp && resp.success) {
            this.$message.success('已保存')
            this.$emit('saved')
          }
        })
        .finally(() => {
          this.saving = false
        })
    }
  }
}
</script>

<style scoped>
.collector-page {
  padding-bottom: 24px;
}
.hint {
  margin-left: 8px;
  color: var(--el-text-color-secondary);
}
.col-head {
  display: inline-flex;
  align-items: center;
}
.addr-tip-icon {
  margin-left: 4px;
  font-size: 14px;
  color: var(--el-text-color-secondary);
  cursor: help;
  vertical-align: middle;
}
.mb-12 {
  margin-bottom: 12px;
}
.mb-16 {
  margin-bottom: 16px;
}
.mt-8 {
  margin-top: 8px;
}
.collector-actions {
  margin-top: 16px;
}
.collector-table :deep(.el-input-number),
.collector-table :deep(.el-select),
.collector-table :deep(.el-input) {
  width: 100%;
}
</style>
