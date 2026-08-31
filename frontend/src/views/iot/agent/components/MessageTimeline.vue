<template>
  <div class="agent-timeline">
    <div v-for="m in messages" :key="m.id" class="tl-row">
      <div v-if="m.role === 'user'" class="bubble-row is-user">
        <div class="bubble user-bubble">
          <div class="bubble-text">{{ m.content }}</div>
        </div>
        <div class="avatar user-avatar">我</div>
      </div>

      <div
        v-else-if="m.eventType === 'confirm_required'"
        class="confirm-card"
        :class="'is-' + confirmStatus(m)"
      >
        <div class="confirm-head">
          <el-tag :type="confirmAlertType(m)" effect="light" round>{{ confirmTitle(m) }}</el-tag>
          <span class="confirm-tool">{{ toolLabel(m.toolName) }}</span>
        </div>
        <el-alert
          v-if="previewMeta(m).warning || previewMeta(m).published"
          type="warning"
          :closable="false"
          show-icon
          class="preview-alert"
          :title="previewMeta(m).warning || $t('agent.previewPublished')"
        />
        <el-alert
          v-if="previewMeta(m).truncated"
          type="info"
          :closable="false"
          show-icon
          class="preview-alert"
          :title="$t('agent.previewTruncated')"
        />
        <div v-if="previewEntries(m).length" class="preview-grid">
          <div v-for="row in previewEntries(m)" :key="row.key" class="preview-row">
            <span class="preview-key">{{ row.label }}</span>
            <span class="preview-val">{{ row.value }}</span>
          </div>
        </div>
        <div v-for="block in docBlocks(m)" :key="m.id + '-' + block.kind" class="doc-block">
          <div class="doc-head">
            <div class="doc-meta">
              <span class="doc-title">{{ block.title }}</span>
              <span v-if="block.summary" class="doc-summary">{{ block.summary }}</span>
            </div>
            <el-button link type="primary" @click="toggleDoc(m, block.kind)">
              {{ isDocOpen(m, block.kind) ? block.hideLabel : block.viewLabel }}
            </el-button>
          </div>
          <div v-if="isDocOpen(m, block.kind)" class="doc-body">
            <MdPreview
              class="doc-md"
              :editorId="'agent-doc-' + m.id + '-' + block.kind"
              :modelValue="block.md"
              :codeFoldable="false"
              previewTheme="github"
            />
          </div>
        </div>
        <pre v-if="!previewEntries(m).length && !docBlocks(m).length" class="preview-raw">{{
          previewText(m)
        }}</pre>
        <div v-if="canAct(m)" class="confirm-actions">
          <el-button type="primary" @click="$emit('apply', m)">{{ $t('agent.apply') }}</el-button>
          <el-button @click="$emit('reject', m)">{{ $t('agent.reject') }}</el-button>
        </div>
      </div>

      <div v-else-if="m.eventType === 'applied'" class="event-chip is-applied">
        <Icon icon="ant-design:check-circle-filled" />
        <span>已应用 {{ toolLabel(m.toolName) }}{{ m.productId ? ' · ' + m.productId : '' }}</span>
      </div>
      <div v-else-if="m.eventType === 'rejected'" class="event-chip is-rejected">
        <Icon icon="ant-design:close-circle-filled" />
        <span>已拒绝 {{ toolLabel(m.toolName) }}</span>
      </div>
      <div v-else-if="m.eventType === 'undone'" class="event-chip is-undone">
        <Icon icon="ant-design:undo-outlined" />
        <span>已撤销 {{ toolLabel(m.toolName) }}</span>
      </div>

      <div v-else-if="m.role === 'assistant' && m.content" class="bubble-row is-assistant">
        <div class="avatar assistant-avatar">
          <Icon icon="ant-design:robot-outlined" :size="16" />
        </div>
        <div class="bubble assistant-bubble">
          <MdPreview
            class="md-body"
            :modelValue="m.content"
            :codeFoldable="false"
            previewTheme="github"
          />
        </div>
      </div>

      <div v-else-if="m.role === 'tool'" class="tool-chip">
        <Icon icon="ant-design:tool-outlined" />
        <span>{{ toolLabel(m.toolName) }}</span>
      </div>
    </div>

    <div v-if="runStatus === 'running' || runStatus === 'applying'" class="bubble-row is-assistant">
      <div class="avatar assistant-avatar">
        <Icon icon="ant-design:robot-outlined" :size="16" />
      </div>
      <div class="bubble assistant-bubble thinking">
        <span class="dot"></span><span class="dot"></span><span class="dot"></span>
        <span class="thinking-text">{{
          runStatus === 'applying' ? $t('agent.applying') : $t('agent.thinking')
        }}</span>
      </div>
    </div>
  </div>
</template>

<script>
import { MdPreview } from 'md-editor-v3'
import 'md-editor-v3/lib/style.css'

const TOOL_LABELS = {
  list_products: '查询产品',
  get_product: '读取产品',
  get_tsl_schema: '物模型约束',
  get_codec_contract: '编解码约定',
  validate_tsl: '校验物模型',
  validate_script: '校验脚本',
  create_product: '创建产品',
  update_product: '更新产品',
  save_tsl: '保存物模型',
  save_script: '保存编解码',
  update_network: '更新网络'
}

const FIELD_LABELS = {
  id: '产品 ID',
  name: '名称',
  networkType: '网络类型',
  productId: '产品',
  desc: '说明',
  tsl: '物模型',
  script: '脚本'
}

const SKIP_SUMMARY = new Set([
  'tsl',
  'script',
  'configuration',
  'truncated',
  'old',
  'new',
  'warning',
  'published'
])

function parseMaybeJson(v) {
  if (typeof v !== 'string') return v
  const s = v.trim()
  if (!s) return v
  try {
    return JSON.parse(s)
  } catch {
    return v
  }
}

function prettyJson(v) {
  const val = parseMaybeJson(v)
  if (typeof val === 'string') return val
  try {
    return JSON.stringify(val, null, 2)
  } catch {
    return String(val)
  }
}

function fence(lang, body) {
  return '```' + lang + '\n' + (body || '') + '\n```'
}

export default {
  name: 'MessageTimeline',
  components: { MdPreview },
  props: {
    messages: { type: Array, default: () => [] },
    runStatus: { type: String, default: 'idle' },
    pendingDrafts: { type: Array, default: () => [] }
  },
  emits: ['apply', 'reject'],
  data() {
    return { openDocs: {} }
  },
  methods: {
    toolLabel(name) {
      return TOOL_LABELS[name] || name || '操作'
    },
    fieldLabel(key) {
      return FIELD_LABELS[key] || key
    },
    previewRoot(m) {
      const p = m && m.preview
      if (!p || typeof p !== 'object' || Array.isArray(p)) {
        return { old: null, neu: null, truncated: false, warning: '', published: false }
      }
      if ('new' in p || 'old' in p) {
        return {
          old: parseMaybeJson(p.old),
          neu: parseMaybeJson(p.new),
          truncated: !!p.truncated,
          warning: p.warning || '',
          published: !!p.published
        }
      }
      return { old: null, neu: p, truncated: false, warning: '', published: false }
    },
    previewMeta(m) {
      const r = this.previewRoot(m)
      return { truncated: r.truncated, warning: r.warning, published: r.published }
    },
    previewText(m) {
      const p = m.preview
      if (!p) return ''
      try {
        return JSON.stringify(p, null, 2)
      } catch {
        return String(p)
      }
    },
    previewEntries(m) {
      const src = this.previewRoot(m).neu
      if (!src || typeof src !== 'object' || Array.isArray(src)) return []
      return Object.keys(src)
        .filter((k) => !SKIP_SUMMARY.has(k) && src[k] != null && src[k] !== '')
        .map((k) => {
          const v = src[k]
          let value = v
          if (typeof v === 'object') {
            try {
              value = JSON.stringify(v)
            } catch {
              value = String(v)
            }
          }
          if (String(value).length > 160) value = String(value).slice(0, 157) + '…'
          return { key: k, label: this.fieldLabel(k), value: String(value) }
        })
    },
    tslSummary(val) {
      const tsl = parseMaybeJson(val)
      if (!tsl || typeof tsl !== 'object') return ''
      const props = Array.isArray(tsl.properties) ? tsl.properties.length : 0
      const funcs = Array.isArray(tsl.functions) ? tsl.functions.length : 0
      const events = Array.isArray(tsl.events) ? tsl.events.length : 0
      return this.$t('agent.tslSummary', { props, funcs, events })
    },
    scriptSummary(val) {
      const text = typeof val === 'string' ? val : prettyJson(val)
      const lines = text ? text.split('\n').length : 0
      return this.$t('agent.scriptSummary', { n: lines })
    },
    docMd(lang, neu, oldVal) {
      if (oldVal != null && oldVal !== '') {
        return (
          '**' +
          this.$t('agent.previewOld') +
          '**\n\n' +
          fence(lang, lang === 'javascript' ? String(oldVal) : prettyJson(oldVal)) +
          '\n\n**' +
          this.$t('agent.previewNew') +
          '**\n\n' +
          fence(lang, lang === 'javascript' ? String(neu) : prettyJson(neu))
        )
      }
      return fence(lang, lang === 'javascript' ? String(neu ?? '') : prettyJson(neu))
    },
    docBlocks(m) {
      const { neu, old } = this.previewRoot(m)
      if (!neu || typeof neu !== 'object' || Array.isArray(neu)) return []
      const oldObj = old && typeof old === 'object' && !Array.isArray(old) ? old : null
      const specs = [
        {
          kind: 'tsl',
          lang: 'json',
          title: this.$t('agent.docTsl'),
          viewLabel: this.$t('agent.viewTsl'),
          hideLabel: this.$t('agent.hideTsl'),
          summary: (v) => this.tslSummary(v)
        },
        {
          kind: 'script',
          lang: 'javascript',
          title: this.$t('agent.docScript'),
          viewLabel: this.$t('agent.viewScript'),
          hideLabel: this.$t('agent.hideScript'),
          summary: (v) => this.scriptSummary(v)
        },
        {
          kind: 'configuration',
          lang: 'json',
          title: this.$t('agent.docConfig'),
          viewLabel: this.$t('agent.viewConfig'),
          hideLabel: this.$t('agent.hideConfig'),
          summary: () => ''
        }
      ]
      return specs
        .filter((s) => neu[s.kind] != null && neu[s.kind] !== '')
        .map((s) => ({
          kind: s.kind,
          title: s.title,
          viewLabel: s.viewLabel,
          hideLabel: s.hideLabel,
          summary: s.summary(neu[s.kind]),
          md: this.docMd(s.lang, neu[s.kind], oldObj ? oldObj[s.kind] : null)
        }))
    },
    docKey(m, kind) {
      return (m.id || m.draftId || '') + ':' + kind
    },
    isDocOpen(m, kind) {
      return this.openDocs[this.docKey(m, kind)] !== false
    },
    toggleDoc(m, kind) {
      const key = this.docKey(m, kind)
      this.openDocs = { ...this.openDocs, [key]: this.openDocs[key] === false }
    },
    confirmStatus(m) {
      return m.draftStatus || 'pending'
    },
    confirmTitle(m) {
      const status = this.confirmStatus(m)
      if (status === 'applied') return this.$t('agent.applied')
      if (status === 'rejected') return this.$t('agent.rejected')
      return this.$t('agent.awaiting')
    },
    confirmAlertType(m) {
      const status = this.confirmStatus(m)
      if (status === 'applied') return 'success'
      if (status === 'rejected') return 'info'
      return 'warning'
    },
    canAct(m) {
      const pending = (this.pendingDrafts || []).some((d) => (d.id || d) === m.draftId)
      return this.confirmStatus(m) === 'pending' && this.runStatus === 'awaiting_confirm' && pending
    }
  }
}
</script>

<style lang="less" scoped>
.agent-timeline {
  padding: 8px 4px 16px;
}

.tl-row {
  margin-bottom: 14px;
}

.bubble-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  &.is-user {
    justify-content: flex-end;
  }
}

.avatar {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 600;
}

.user-avatar {
  color: #fff;
  background: var(--el-color-primary);
}

.assistant-avatar {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}

.bubble {
  max-width: min(720px, 78%);
  border-radius: 12px;
  line-height: 1.6;
  word-break: break-word;
}

.user-bubble {
  background: var(--el-color-primary);
  color: #fff;
  padding: 10px 14px;
  border-bottom-right-radius: 4px;
}

.bubble-text {
  white-space: pre-wrap;
  font-size: 14px;
}

.assistant-bubble {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  padding: 4px 12px 2px;
  border-bottom-left-radius: 4px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
}

.thinking {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 10px 14px;
  color: var(--el-text-color-secondary);
}

.thinking-text {
  font-size: 13px;
}

.dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-color-primary);
  opacity: 0.35;
  animation: blink 1.2s infinite ease-in-out;
  &:nth-child(2) {
    animation-delay: 0.15s;
  }
  &:nth-child(3) {
    animation-delay: 0.3s;
  }
}

@keyframes blink {
  0%,
  80%,
  100% {
    opacity: 0.25;
    transform: translateY(0);
  }
  40% {
    opacity: 1;
    transform: translateY(-2px);
  }
}

.md-body {
  background: transparent !important;
}

:deep(.md-editor) {
  background: transparent;
}
:deep(.md-editor-preview-wrapper) {
  padding: 4px 0;
}
:deep(.md-editor-preview) {
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.confirm-card {
  margin: 4px 42px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-left: 3px solid var(--el-color-warning);
  border-radius: 10px;
  padding: 12px 14px;
  &.is-applied {
    border-left-color: var(--el-color-success);
  }
  &.is-rejected {
    border-left-color: var(--el-color-info);
  }
}

.confirm-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
}

.confirm-tool {
  font-size: 13px;
  color: var(--el-text-color-regular);
  font-weight: 600;
}

.preview-alert {
  margin-bottom: 10px;
}

.doc-block {
  margin-top: 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
  background: var(--el-fill-color-blank);
}

.doc-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
}

.doc-meta {
  display: flex;
  align-items: baseline;
  gap: 10px;
  min-width: 0;
}

.doc-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.doc-summary {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.doc-body {
  max-height: 360px;
  overflow: auto;
  border-top: 1px solid var(--el-border-color-extra-light);
  padding: 0 8px 8px;
}

.doc-md {
  background: transparent !important;
}

.doc-md :deep(.md-editor) {
  background: transparent;
}

.doc-md :deep(.md-editor-preview-wrapper) {
  padding: 8px 4px;
}

.doc-md :deep(pre) {
  margin: 0;
}

.preview-grid {
  display: grid;
  gap: 6px;
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 10px 12px;
}

.preview-row {
  display: grid;
  grid-template-columns: 88px 1fr;
  gap: 8px;
  font-size: 13px;
}

.preview-key {
  color: var(--el-text-color-secondary);
}

.preview-val {
  color: var(--el-text-color-primary);
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
  word-break: break-all;
}

.preview-raw {
  margin: 0;
  background: var(--el-fill-color-light);
  padding: 10px 12px;
  border-radius: 8px;
  overflow: auto;
  max-height: 240px;
  font-size: 12px;
}

.confirm-actions {
  margin-top: 12px;
  display: flex;
  gap: 8px;
}

.event-chip,
.tool-chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-left: 42px;
  padding: 4px 10px;
  border-radius: 999px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}

.event-chip.is-applied {
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
}

.event-chip.is-rejected {
  color: var(--el-color-info);
  background: var(--el-fill-color);
}

.event-chip.is-undone {
  color: var(--el-color-warning);
  background: var(--el-color-warning-light-9);
}

@media (max-width: 768px) {
  .bubble {
    max-width: 88%;
  }
  .confirm-card,
  .event-chip,
  .tool-chip {
    margin-left: 0;
    margin-right: 0;
  }
  .preview-row {
    grid-template-columns: 1fr;
    gap: 2px;
  }
}
</style>
