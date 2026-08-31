<template>
  <el-drawer
    v-if="visible"
    :title="`调试日志(${productId})`"
    placement="right"
    :model-value="true"
    :close-on-click-modal="false"
    size="70%"
    @close="close"
  >
    <div class="debug-wrap">
      <div class="debug-toolbar">
        <div class="debug-status">
          <span class="debug-dot" :class="{ on: isConnect && !paused, paused: paused }"></span>
          <span>{{ statusText }}</span>
          <span v-if="paused && dropped" class="debug-dropped">已丢弃 {{ dropped }} 条</span>
        </div>
        <div class="debug-filters">
          <el-select v-model="filterLevel" size="small" class="debug-level-select">
            <el-option label="全部级别" value="all" />
            <el-option label="DEBUG" value="debug" />
            <el-option label="INFO" value="info" />
            <el-option label="WARN" value="warn" />
            <el-option label="ERROR" value="error" />
          </el-select>
          <el-input
            v-model="filterText"
            size="small"
            clearable
            placeholder="输入内容过滤"
            class="debug-filter-input"
          />
        </div>
        <div class="debug-actions">
          <el-button size="small" @click="togglePause">
            {{ paused ? '继续接收' : '暂停接收' }}
          </el-button>
          <el-button size="small" @click="refresh">刷新</el-button>
          <el-button size="small" @click="clearLogs">清空</el-button>
        </div>
      </div>
      <div class="product-debug" :class="{ isConnect: isConnect, isPaused: paused }">
        <div ref="aceEl" class="debug-ace"></div>
      </div>
      <div v-if="hasNewWhileAway" class="debug-jump" @click="jumpToBottom">
        有新日志，点击回到底部
      </div>
    </div>
  </el-drawer>
</template>

<script>
import { getEventBusUrl } from '@/views/iot/product/api.js'
import { EventBusWs } from '@/utils/eventBusWs'
import ace from 'ace-builds'
import dayjs from 'dayjs'
import { createIotLogMode, escapeRegExp } from './iotLogMode.js'

import 'ace-builds/src-noconflict/theme-tomorrow_night'
import 'ace-builds/src-noconflict/mode-text'

const MAX_ITEMS = 2000
const FLUSH_MS = 80

export default {
  name: 'ProductDebugLog',
  props: {
    productId: {
      type: String,
      required: true
    }
  },
  data() {
    return {
      visible: false,
      paused: false,
      dropped: 0,
      autoScroll: true,
      hasNewWhileAway: false,
      ignoreScroll: false,
      filterLevel: 'all',
      filterText: '',
      isConnect: false
    }
  },
  computed: {
    statusText() {
      if (this.paused) {
        return '已暂停接收'
      }
      return this.isConnect ? '已连接' : '未连接'
    }
  },
  watch: {
    filterLevel() {
      this.rebuildAce()
    },
    filterText() {
      if (this._filterTimer) {
        clearTimeout(this._filterTimer)
      }
      this._filterTimer = setTimeout(() => {
        this._filterTimer = null
        this.rebuildAce()
      }, 150)
    }
  },
  created() {
    this._logs = []
    this._pending = []
    this._seq = 0
    this._flushTimer = null
    this._filterTimer = null
    this._ace = null
    this._ws = null
  },
  beforeUnmount() {
    this.teardown()
  },
  methods: {
    open() {
      this._logs = []
      this._pending = []
      this._seq = 0
      this.paused = false
      this.dropped = 0
      this.autoScroll = true
      this.hasNewWhileAway = false
      this.filterLevel = 'all'
      this.filterText = ''
      this.visible = true
      this.connectWs()
      this.mountAce()
    },
    close() {
      this.visible = false
      this.teardown()
    },
    teardown() {
      this.clearTimers()
      this.destroyAce()
      if (this._ws) {
        this._ws.close()
        this._ws = null
      }
    },
    connectWs() {
      if (this._ws) {
        this._ws.close()
        this._ws = null
      }
      this._ws = new EventBusWs(
        getEventBusUrl(this.productId, '*', 'debug'),
        (evt) => this.onMessage(evt),
        (connected) => this.onStatus(connected)
      )
      this._ws.connect()
    },
    refresh() {
      this.connectWs()
      this.autoScroll = true
      this.hasNewWhileAway = false
      this.scrollToBottom()
    },
    mountAce() {
      this.$nextTick(() => {
        const el = this.$refs.aceEl
        if (!el) {
          return
        }
        if (this._ace) {
          this._ace.resize(true)
          return
        }
        const editor = ace.edit(el)
        editor.setTheme('ace/theme/tomorrow_night')
        editor.session.setMode(createIotLogMode())
        editor.setOptions({
          readOnly: true,
          wrap: true,
          useWorker: false,
          showPrintMargin: false,
          highlightActiveLine: false,
          highlightGutterLine: false,
          showLineNumbers: true,
          fontSize: 13,
          tabSize: 2
        })
        editor.renderer.setShowGutter(true)
        editor.session.setOption('firstLineNumber', 1)
        editor.session.on('changeScrollTop', this.onAceScroll)
        this._ace = editor
        this.rebuildAce()
        setTimeout(() => editor.resize(true), 50)
        setTimeout(() => editor.resize(true), 320)
      })
    },
    destroyAce() {
      if (this._ace) {
        this._ace.destroy()
        this._ace = null
      }
      const el = this.$refs.aceEl
      if (el) {
        el.innerHTML = ''
      }
    },
    clearTimers() {
      if (this._flushTimer) {
        clearTimeout(this._flushTimer)
        this._flushTimer = null
      }
      if (this._filterTimer) {
        clearTimeout(this._filterTimer)
        this._filterTimer = null
      }
    },
    onMessage(evt) {
      if (this.paused) {
        this.dropped += 1
        return
      }
      let data
      try {
        data = JSON.parse(evt.data)
      } catch (e) {
        data = { createTime: new Date(), productId: this.productId, data: evt.data }
      }
      this.enqueue(data)
    },
    onStatus(connected) {
      this.isConnect = connected
      this.enqueue({
        createTime: new Date(),
        productId: this.productId,
        data: connected ? '已连接' : '连接关闭'
      })
    },
    enqueue(raw) {
      const text = this.formatData(raw?.data)
      const deviceId = raw?.deviceId && raw.deviceId !== '-' ? raw.deviceId : ''
      this._pending.push({
        seq: ++this._seq,
        timeText: this.formatTime(raw?.createTime),
        deviceId,
        level: this.resolveLevel(raw?.level, text),
        text
      })
      if (!this._flushTimer) {
        this._flushTimer = setTimeout(() => this.flush(), FLUSH_MS)
      }
    },
    flush() {
      this._flushTimer = null
      const batch = this._pending
      if (!batch.length) {
        return
      }
      this._pending = []
      this._logs.push(...batch)
      let removed = []
      if (this._logs.length > MAX_ITEMS) {
        removed = this._logs.splice(0, this._logs.length - MAX_ITEMS)
      }
      if (removed.length) {
        let removedVisible = 0
        for (let i = 0; i < removed.length; i++) {
          if (this.matchItem(removed[i])) {
            removedVisible += 1
          }
        }
        this.removeAceHeadLines(removedVisible)
      }
      const lines = []
      for (let i = 0; i < batch.length; i++) {
        if (this.matchItem(batch[i])) {
          lines.push(this.formatLine(batch[i]))
        }
      }
      if (lines.length) {
        if (!this.autoScroll) {
          this.hasNewWhileAway = true
        }
        this.appendAceText(lines.join('\n'))
        this.applyFilterHighlight()
        this.scrollToBottom()
      } else if (removed.length && this.autoScroll) {
        this.scrollToBottom()
      }
      if (removed.length || lines.length) {
        this.syncAceFirstLineNumber()
      }
    },
    matchItem(item) {
      if (this.filterLevel !== 'all' && item.level !== this.filterLevel) {
        return false
      }
      const keyword = this.filterText.trim().toLowerCase()
      if (!keyword) {
        return true
      }
      return (
        item.text.toLowerCase().includes(keyword) ||
        item.timeText.toLowerCase().includes(keyword) ||
        (item.deviceId && item.deviceId.toLowerCase().includes(keyword)) ||
        item.level.includes(keyword)
      )
    },
    formatLine(item) {
      const device = item.deviceId ? `[${item.deviceId}] ` : ''
      return `${item.timeText} [${item.level.toUpperCase()}] ${device}${item.text}`
    },
    rebuildAce() {
      this.flush()
      const editor = this._ace
      if (!editor) {
        return
      }
      const lines = []
      const logs = this._logs
      for (let i = 0; i < logs.length; i++) {
        if (this.matchItem(logs[i])) {
          lines.push(this.formatLine(logs[i]))
        }
      }
      this.ignoreScroll = true
      editor.session.setValue(lines.join('\n'))
      this.syncAceFirstLineNumber()
      this.applyFilterHighlight()
      this.scrollToBottom()
    },
    appendAceText(text) {
      const editor = this._ace
      if (!editor) {
        return
      }
      const session = editor.session
      const lastRow = session.getLength() - 1
      const lastLine = session.getLine(lastRow)
      const prefix = lastLine ? '\n' : ''
      session.insert({ row: lastRow, column: lastLine.length }, prefix + text)
    },
    removeAceHeadLines(count) {
      const editor = this._ace
      if (!editor || count <= 0) {
        return
      }
      const session = editor.session
      const len = session.getLength()
      if (count >= len) {
        session.setValue('')
        return
      }
      session.remove({
        start: { row: 0, column: 0 },
        end: { row: count, column: 0 }
      })
    },
    syncAceFirstLineNumber() {
      const editor = this._ace
      if (!editor) {
        return
      }
      const first = this.firstVisibleLog()
      const firstNumber = first ? first.seq : Math.max(this._seq, 0) + 1
      editor.session.setOption('firstLineNumber', firstNumber)
    },
    firstVisibleLog() {
      const logs = this._logs
      for (let i = 0; i < logs.length; i++) {
        if (this.matchItem(logs[i])) {
          return logs[i]
        }
      }
      return null
    },
    applyFilterHighlight() {
      const editor = this._ace
      if (!editor) {
        return
      }
      const keyword = this.filterText.trim()
      const re = keyword ? new RegExp(escapeRegExp(keyword), 'gi') : null
      editor.session.highlight(re)
      editor.renderer.updateBackMarkers()
    },
    resolveLevel(level, text) {
      const normalized = String(level || '')
        .trim()
        .toLowerCase()
      if (['debug', 'info', 'warn', 'warning', 'error', 'log'].includes(normalized)) {
        if (normalized === 'warning') {
          return 'warn'
        }
        if (normalized === 'log') {
          return 'debug'
        }
        return normalized
      }
      const s = String(text || '')
      if (s === '已连接' || s === '连接关闭') {
        return 'info'
      }
      if (/(error|exception|panic|失败|错误|denied)/i.test(s)) {
        return 'error'
      }
      if (/(warn(?:ing)?|警告)/i.test(s)) {
        return 'warn'
      }
      if (/(^|[\s:])info([\s:]|$)/i.test(s)) {
        return 'info'
      }
      return 'debug'
    },
    formatTime(value) {
      if (value == null || value === '') {
        return ''
      }
      if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2} /.test(value)) {
        return value
      }
      const d = dayjs(value)
      return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss.SSS') : String(value)
    },
    formatData(data) {
      if (data == null) {
        return ''
      }
      if (typeof data === 'object') {
        try {
          return JSON.stringify(data)
        } catch (e) {
          return String(data)
        }
      }
      return String(data)
    },
    togglePause() {
      this.paused = !this.paused
      if (this.paused) {
        this.dropped = 0
        this.flush()
      } else {
        this.autoScroll = true
        this.hasNewWhileAway = false
        this.scrollToBottom()
      }
    },
    clearLogs() {
      this._logs = []
      this._pending = []
      this._seq = 0
      this.dropped = 0
      this.autoScroll = true
      this.hasNewWhileAway = false
      if (this._ace) {
        this._ace.session.setValue('')
        this._ace.session.setOption('firstLineNumber', 1)
      }
    },
    onAceScroll() {
      if (this.ignoreScroll || !this._ace) {
        return
      }
      const editor = this._ace
      const lastVisible = editor.renderer.getLastVisibleRow()
      const last = editor.session.getLength() - 1
      this.autoScroll = lastVisible >= last - 3
      if (this.autoScroll) {
        this.hasNewWhileAway = false
      }
    },
    scrollToBottom() {
      if (!this.autoScroll || !this._ace) {
        return
      }
      const editor = this._ace
      const last = Math.max(editor.session.getLength() - 1, 0)
      this.ignoreScroll = true
      editor.scrollToLine(last, false, false)
      requestAnimationFrame(() => {
        this.ignoreScroll = false
      })
    },
    jumpToBottom() {
      this.autoScroll = true
      this.hasNewWhileAway = false
      this.scrollToBottom()
    }
  }
}
</script>

<style lang="less" scoped>
.debug-wrap {
  position: relative;
  display: flex;
  flex-direction: column;
  height: calc(100vh - 114px);
}
.debug-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: 8px 12px;
  padding: 0 0 10px;
  flex-shrink: 0;
}
.debug-filters {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 240px;
}
.debug-level-select {
  width: 118px;
}
.debug-filter-input {
  flex: 1;
  min-width: 160px;
  max-width: 280px;
}
.debug-status {
  display: flex;
  align-items: center;
  gap: 8px;
  color: #606266;
  font-size: 13px;
}
.debug-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #f56c6c;
  flex-shrink: 0;
  &.on {
    background: #52c41a;
  }
  &.paused {
    background: #e6a23c;
  }
}
.debug-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}
.debug-dropped {
  color: #e6a23c;
}
.product-debug {
  flex: 1;
  min-height: 0;
  width: 100%;
  border-top: 4px solid #f56c6c;
  box-sizing: border-box;
  overflow: hidden;
  &.isConnect {
    border-top-color: #52c41a;
  }
  &.isPaused {
    border-top-color: #e6a23c;
  }
}
.debug-ace {
  position: relative;
  height: 100%;
  width: 100%;
  :deep(.ace_log-time) {
    color: #67c23a;
  }
  :deep(.ace_log-level-debug) {
    color: #909399;
  }
  :deep(.ace_log-level-info) {
    color: #409eff;
    font-weight: 600;
  }
  :deep(.ace_log-level-warn) {
    color: #e6a23c;
    font-weight: 600;
  }
  :deep(.ace_log-level-error) {
    color: #f56c6c;
    font-weight: 600;
  }
  :deep(.ace_log-device) {
    color: #56b6c2;
  }
  :deep(.ace_log-json-key) {
    color: #c678dd;
  }
  :deep(.ace_string) {
    color: #98c379;
  }
  :deep(.ace_constant) {
    color: #d19a66;
  }
  :deep(.ace_selected-word) {
    background: rgba(64, 158, 255, 0.35);
    border: 0;
    border-radius: 2px;
  }
}
.debug-jump {
  position: absolute;
  right: 24px;
  bottom: 16px;
  padding: 6px 12px;
  background: rgba(64, 158, 255, 0.92);
  color: #fff;
  border-radius: 14px;
  font-size: 12px;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.2);
  z-index: 2;
}
</style>
