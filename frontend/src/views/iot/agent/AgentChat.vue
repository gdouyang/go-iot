<template>
  <div class="agent-page">
    <ContentWrap title="AI 助手">
      <div class="agent-shell">
        <aside class="agent-aside">
          <div class="aside-head">
            <span class="aside-title">会话</span>
            <el-button type="primary" size="small" :disabled="!modelConfigured" @click="newChat">
              <Icon icon="ant-design:plus-outlined" />
              {{ $t('agent.newChat') }}
            </el-button>
          </div>
          <el-empty
            v-if="!convs.length"
            class="aside-empty"
            :image-size="56"
            :description="$t('agent.noConversations')"
          />
          <div v-else class="conv-list">
            <div
              v-for="c in convs"
              :key="c.id"
              class="conv-item"
              :class="{ active: c.id === currentConvId }"
              @click="selectConv(c.id)"
            >
              <div class="conv-icon">
                <Icon icon="ant-design:message-outlined" />
              </div>
              <div class="conv-meta">
                <div class="conv-title" :title="c.title || c.id" @dblclick.stop="startRename(c)">
                  {{ c.title || c.id }}
                </div>
                <div class="conv-time">{{ formatTime(c.updateTime || c.createTime) }}</div>
              </div>
              <div class="conv-actions" @click.stop>
                <el-button
                  class="conv-act"
                  link
                  :title="$t('agent.renameConv')"
                  @click="startRename(c)"
                >
                  <Icon icon="ant-design:edit-outlined" />
                </el-button>
                <el-popconfirm
                  :title="$t('agent.deleteConvConfirm')"
                  width="220"
                  @confirm="removeConv(c)"
                >
                  <template #reference>
                    <el-button class="conv-act" link type="danger" :title="$t('agent.deleteConv')">
                      <Icon icon="ant-design:delete-outlined" />
                    </el-button>
                  </template>
                </el-popconfirm>
              </div>
            </div>
          </div>
        </aside>

        <section class="agent-main">
          <header class="main-head">
            <div class="head-left">
              <div
                class="head-title"
                :title="$t('agent.renameConv')"
                @dblclick="currentConv && startRename(currentConv)"
              >
                {{ currentTitle }}
              </div>
              <el-tag
                v-if="currentConvId && statusMeta.text"
                :type="statusMeta.type"
                effect="plain"
                round
                size="small"
              >
                {{ statusMeta.text }}
              </el-tag>
            </div>
            <el-button @click="openConfig">
              <Icon icon="ant-design:setting-outlined" />
              {{ $t('agent.config') }}
            </el-button>
          </header>

          <el-alert
            v-if="!modelConfigured"
            type="warning"
            :closable="false"
            :title="$t('agent.notConfigured')"
            class="cfg-alert"
            show-icon
          >
            <el-button type="primary" link @click="openConfig">{{
              $t('agent.goConfigure')
            }}</el-button>
          </el-alert>
          <el-alert
            v-else-if="canContinue"
            type="success"
            :closable="false"
            :title="$t('agent.resumeHint')"
            class="cfg-alert"
            show-icon
          >
            <el-button type="primary" link @click="onContinue">{{
              $t('agent.continue')
            }}</el-button>
          </el-alert>

          <div ref="timelineEl" class="timeline-wrap">
            <el-empty v-if="!currentConvId" :description="$t('agent.selectOrCreate')">
              <el-button type="primary" :disabled="!modelConfigured" @click="newChat">
                {{ $t('agent.newChat') }}
              </el-button>
            </el-empty>
            <el-empty
              v-else-if="!messages.length && runStatus !== 'running'"
              :description="$t('agent.emptyChat')"
            />
            <MessageTimeline
              v-else
              :messages="messages"
              :run-status="runStatus"
              :pending-drafts="pendingDrafts"
              @apply="onApply"
              @reject="onReject"
            />
          </div>

          <footer v-if="currentConvId" class="composer">
            <el-input
              v-model="draft"
              type="textarea"
              :rows="3"
              resize="none"
              :disabled="!canSend"
              :placeholder="$t('agent.composerPlaceholder')"
              @keydown="onComposerKeydown"
              @compositionstart="onComposeStart"
              @compositionend="onComposeEnd"
            />
            <div class="composer-bar">
              <span class="composer-hint">{{ $t('agent.composerHint') }}</span>
              <div class="composer-actions">
                <el-button v-if="canContinue" @click="onContinue">{{
                  $t('agent.continue')
                }}</el-button>
                <el-button
                  type="primary"
                  :disabled="!canSend || !draft.trim()"
                  :loading="sending"
                  @click="send"
                >
                  <Icon v-if="!sending" icon="ant-design:send-outlined" />
                  {{ $t('agent.send') }}
                </el-button>
              </div>
            </div>
          </footer>
        </section>
      </div>
    </ContentWrap>
    <AgentModelDialog ref="modelDialogRef" @confirm="onSettingsSaved" />
    <el-dialog
      v-model="renameVisible"
      :title="$t('agent.renameConv')"
      width="420px"
      append-to-body
      @opened="focusRenameInput"
      @closed="cancelRename"
    >
      <el-input
        ref="renameInput"
        v-model="editTitle"
        maxlength="128"
        show-word-limit
        :placeholder="$t('agent.renamePlaceholder')"
        @keydown="onRenameKeydown"
      />
      <template #footer>
        <el-button @click="renameVisible = false">{{ $t('common.cancel') }}</el-button>
        <el-button
          type="primary"
          :disabled="!editTitle.trim()"
          :loading="renaming"
          @click="saveRename"
        >
          {{ $t('common.ok') }}
        </el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script>
import { ContentWrap } from '@/components/ContentWrap'
import {
  getAgentStatus,
  pageConversations,
  createConversation,
  getConversation,
  renameConversation,
  pageMessages,
  continueRun,
  applyDrafts,
  rejectDrafts,
  deleteConversation
} from './api.js'
import { postAgentStream } from './stream.js'
import AgentModelDialog from './components/AgentModelDialog.vue'
import MessageTimeline from './components/MessageTimeline.vue'

export default {
  name: 'AgentChat',
  components: { ContentWrap, AgentModelDialog, MessageTimeline },
  data() {
    return {
      modelConfigured: false,
      convs: [],
      currentConvId: '',
      messages: [],
      runStatus: 'idle',
      needsResume: false,
      pendingDrafts: [],
      draft: '',
      sending: false,
      composing: false,
      imeLock: false,
      imeLockTimer: 0,
      renameVisible: false,
      editingId: '',
      editTitle: '',
      renaming: false
    }
  },
  computed: {
    currentConv() {
      return this.convs.find((x) => x.id === this.currentConvId) || null
    },
    currentTitle() {
      const c = this.currentConv
      return (c && (c.title || c.id)) || this.$t('agent.selectOrCreate')
    },
    canSend() {
      return (
        this.modelConfigured &&
        this.currentConvId &&
        !this.sending &&
        this.runStatus !== 'running' &&
        this.runStatus !== 'applying'
      )
    },
    canContinue() {
      return this.canSend && this.needsResume && this.runStatus === 'idle'
    },
    statusMeta() {
      if (this.sending || this.runStatus === 'running') {
        return { text: this.$t('agent.thinking'), type: 'primary' }
      }
      if (this.runStatus === 'applying') {
        return { text: this.$t('agent.applying'), type: 'primary' }
      }
      if (this.runStatus === 'awaiting_confirm') {
        return { text: this.$t('agent.awaiting'), type: 'warning' }
      }
      if (this.currentConvId) {
        return { text: this.$t('agent.ready'), type: 'info' }
      }
      return { text: '', type: 'info' }
    }
  },
  created() {
    this.bootstrap()
  },
  methods: {
    sortConvs() {
      this.convs = this.convs.slice().sort((a, b) => {
        const ta = String(a.updateTime || a.createTime || '')
        const tb = String(b.updateTime || b.createTime || '')
        if (ta !== tb) return ta < tb ? 1 : -1
        return String(b.id || '').localeCompare(String(a.id || ''))
      })
    },
    formatTime(t) {
      if (!t) return ''
      const s = String(t)
      const now = new Date()
      const pad = (n) => String(n).padStart(2, '0')
      const today = `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`
      if (s.startsWith(today) && s.length >= 16) return s.slice(11, 16)
      if (s.length >= 16) return s.slice(5, 16)
      return s
    },
    scrollBottom() {
      this.$nextTick(() => {
        const el = this.$refs.timelineEl
        if (el) el.scrollTop = el.scrollHeight
      })
    },
    bootstrap() {
      getAgentStatus()
        .then((resp) => {
          const r = (resp && resp.result) || {}
          this.modelConfigured = !!r.modelConfigured
          return pageConversations()
        })
        .then((resp) => {
          if (resp && resp.result) {
            this.convs = resp.result.list || []
            this.sortConvs()
          }
        })
        .catch(() => {})
    },
    openConfig() {
      this.$refs.modelDialogRef.open({ title: '配置模型' })
    },
    onSettingsSaved() {
      getAgentStatus().then((resp) => {
        const r = (resp && resp.result) || {}
        this.modelConfigured = !!r.modelConfigured
      })
    },
    newChat() {
      createConversation({ title: '新会话' }).then((resp) => {
        const c = resp.result
        this.convs = [c, ...this.convs]
        this.selectConv(c.id)
      })
    },
    selectConv(id) {
      this.currentConvId = id
      getConversation(id).then((resp) => {
        const r = resp.result || {}
        const conv = r.conversation
        if (conv && conv.id) {
          const i = this.convs.findIndex((x) => x.id === id)
          if (i > -1) {
            this.convs.splice(i, 1, Object.assign({}, this.convs[i], conv))
            this.sortConvs()
          }
        }
        this.runStatus = r.runStatus || (conv && conv.runStatus) || 'idle'
        this.needsResume = !!r.needsResume
        this.pendingDrafts = r.pendingDrafts || []
      })
      pageMessages(id).then((resp) => {
        this.messages = (resp.result && resp.result.list) || []
        this.scrollBottom()
      })
    },
    startRename(c) {
      if (!c || !c.id) return
      this.editingId = c.id
      this.editTitle = c.title || ''
      this.renameVisible = true
    },
    focusRenameInput() {
      const el = this.$refs.renameInput
      const input = el && el.$el ? el.$el.querySelector('input') : null
      if (input) {
        input.focus()
        input.select()
      }
    },
    cancelRename() {
      this.renameVisible = false
      this.editingId = ''
      this.editTitle = ''
    },
    onRenameKeydown(e) {
      if (e.key !== 'Enter' || e.shiftKey || e.isComposing || e.keyCode === 229) return
      e.preventDefault()
      this.saveRename()
    },
    saveRename() {
      const c = this.convs.find((x) => x.id === this.editingId)
      if (!c || this.renaming) return
      const title = (this.editTitle || '').trim()
      if (!title) return
      if (title === (c.title || '').trim()) {
        this.renameVisible = false
        return
      }
      this.renaming = true
      renameConversation(c.id, { title })
        .then((resp) => {
          const updated = (resp && resp.result) || { title }
          const i = this.convs.findIndex((x) => x.id === c.id)
          if (i > -1) {
            this.convs.splice(i, 1, Object.assign({}, this.convs[i], updated))
            this.sortConvs()
          }
          this.renameVisible = false
        })
        .catch((err) => {
          this.runError(err)
        })
        .finally(() => {
          this.renaming = false
        })
    },
    removeConv(c) {
      deleteConversation(c.id).then(() => {
        this.convs = this.convs.filter((x) => x.id !== c.id)
        if (this.currentConvId === c.id) {
          this.currentConvId = ''
          this.messages = []
          this.runStatus = 'idle'
          this.needsResume = false
          this.pendingDrafts = []
        }
      })
    },
    onComposeStart() {
      this.composing = true
    },
    onComposeEnd() {
      this.composing = false
      this.imeLock = true
      clearTimeout(this.imeLockTimer)
      this.imeLockTimer = setTimeout(() => {
        this.imeLock = false
      }, 50)
    },
    onComposerKeydown(e) {
      if (e.key !== 'Enter' || e.shiftKey || e.altKey || e.ctrlKey || e.metaKey) {
        return
      }
      if (this.composing || e.isComposing || e.keyCode === 229 || this.imeLock) {
        this.imeLock = false
        return
      }
      e.preventDefault()
      this.send()
    },
    send() {
      const text = (this.draft || '').trim()
      if (!text || !this.canSend) return
      this.draft = ''
      this.sending = true
      this.runStatus = 'running'
      this.messages = this.messages.concat([
        { id: 'tmp-' + Date.now(), role: 'user', content: text }
      ])
      this.scrollBottom()
      postAgentStream(`/agent/conversations/${this.currentConvId}/messages`, { content: text }, {})
        .then(() => this.selectConv(this.currentConvId))
        .catch((err) => {
          this.runError(err)
          this.selectConv(this.currentConvId)
        })
        .finally(() => {
          this.sending = false
        })
    },
    runError(err) {
      const data = err && err.response && err.response.data
      const reason = data && data.result && data.result.reason
      if (reason === 'model_not_configured') {
        this.modelConfigured = false
        this.openConfig()
        return
      }
      const msg = (data && (data.message || data.msg)) || (err && err.message) || '请求失败'
      this.$message.error(msg)
    },
    onContinue() {
      if (!this.canContinue) return
      this.sending = true
      this.runStatus = 'running'
      continueRun(this.currentConvId)
        .then(() => this.selectConv(this.currentConvId))
        .catch((err) => {
          this.runError(err)
          this.selectConv(this.currentConvId)
        })
        .finally(() => {
          this.sending = false
        })
    },
    onApply(m) {
      this.sending = true
      this.runStatus = 'running'
      applyDrafts(this.currentConvId, { draftIds: [m.draftId], resume: true })
        .then((resp) => {
          const failed = (resp.result && resp.result.failed) || []
          if (failed.length) {
            this.$message.error('部分应用失败')
          }
          this.selectConv(this.currentConvId)
        })
        .catch((err) => {
          this.runError(err)
          this.selectConv(this.currentConvId)
        })
        .finally(() => {
          this.sending = false
        })
    },
    onReject(m) {
      rejectDrafts(this.currentConvId, { draftIds: [m.draftId] }).then(() =>
        this.selectConv(this.currentConvId)
      )
    }
  }
}
</script>

<style lang="less" scoped>
.agent-page {
  height: calc(100vh - 100px);
}

.agent-page :deep(.el-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.agent-page :deep(.el-card__body) {
  flex: 1;
  min-height: 0;
  padding: 0;
  display: flex;
}

.agent-page :deep(.el-card__body) > div {
  flex: 1;
  min-width: 0;
  min-height: 0;
  width: 100%;
  display: flex;
}

.agent-shell {
  display: flex;
  width: 100%;
  min-height: 0;
  flex: 1;
}

.agent-aside {
  width: 260px;
  flex: none;
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}

.aside-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 14px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
}

.aside-title {
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.aside-empty {
  padding-top: 40px;
}

.conv-list {
  flex: 1;
  overflow: auto;
  padding: 8px;
}

.conv-item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 10px;
  border-radius: 10px;
  cursor: pointer;
  margin-bottom: 4px;
  transition: background 0.15s ease;
  &:hover {
    background: var(--el-fill-color-light);
    .conv-actions {
      opacity: 1;
    }
  }
  &.active {
    background: var(--el-color-primary-light-9);
    .conv-icon {
      color: var(--el-color-primary);
      background: var(--el-color-primary-light-8);
    }
    .conv-title {
      color: var(--el-color-primary);
    }
  }
}

.conv-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
  flex: none;
}

.conv-meta {
  min-width: 0;
  flex: 1;
}

.conv-title {
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.conv-time {
  margin-top: 2px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.conv-actions {
  opacity: 0;
  flex: none;
  display: flex;
  align-items: center;
  gap: 0;
}

.conv-act {
  padding: 0 4px;
}

.head-title {
  cursor: pointer;
}

.agent-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  background: var(--el-fill-color-blank);
}

.main-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  background: var(--el-bg-color);
}

.head-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.head-title {
  font-size: 15px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.cfg-alert {
  margin: 12px 16px 0;
}

.timeline-wrap {
  flex: 1;
  min-height: 0;
  overflow: auto;
  padding: 16px 20px;
  background:
    linear-gradient(180deg, var(--el-fill-color-blank) 0, transparent 48px),
    var(--app-content-bg-color);
}

.composer {
  border-top: 1px solid var(--el-border-color-extra-light);
  padding: 12px 16px 14px;
  background: var(--el-bg-color);
}

.composer :deep(.el-textarea__inner) {
  box-shadow: none;
  border-radius: 10px;
  padding: 10px 12px;
  background: var(--el-fill-color-blank);
}

.composer-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 8px;
}

.composer-hint {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

.composer-actions {
  display: flex;
  gap: 8px;
}

@media (max-width: 768px) {
  .agent-page {
    height: auto;
    min-height: calc(100vh - 80px);
  }
  .agent-shell {
    flex-direction: column;
  }
  .agent-aside {
    width: 100%;
    max-height: 168px;
    border-right: none;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }
  .aside-empty {
    padding-top: 4px;
    :deep(.el-empty__image) {
      display: none;
    }
  }
  .timeline-wrap {
    min-height: 320px;
  }
}
</style>
