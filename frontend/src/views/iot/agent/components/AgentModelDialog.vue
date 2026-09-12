<template>
  <div>
    <Dialog ref="dlg" :show-ok="true" :ok-text="'保存'" @confirm="onConfirm" maxHeight="auto">
      <el-form ref="formRef" :model="form" label-width="140px">
        <el-form-item
          label="模型 URL"
          prop="baseUrl"
          :rules="[
            {
              required: true,
              whitespace: true,
              message: '请输入模型 URL',
              trigger: ['blur', 'change']
            }
          ]"
        >
          <el-select
            v-model="form.baseUrl"
            filterable
            allow-create
            default-first-option
            :placeholder="placeholder.baseUrl || 'https://api.deepseek.com'"
            style="width: 100%"
            @change="onBaseUrlChange"
          >
            <el-option
              v-for="p in presets"
              :key="p.id"
              :label="p.label + ' · ' + p.baseUrl"
              :value="p.baseUrl"
            />
          </el-select>
        </el-form-item>
        <el-form-item
          label="模型"
          prop="model"
          :rules="[
            { required: true, whitespace: true, message: '请输入模型', trigger: ['blur', 'change'] }
          ]"
        >
          <el-select
            v-model="form.model"
            filterable
            allow-create
            default-first-option
            :placeholder="placeholder.model || 'deepseek-flash'"
            style="width: 100%"
          >
            <el-option v-for="m in modelOptions" :key="m" :label="m" :value="m" />
          </el-select>
        </el-form-item>
        <el-form-item
          label="API Key"
          prop="apiKey"
          :rules="apiKeySet ? [] : [{ required: true, message: '请输入 API Key' }]"
        >
          <el-input
            v-model="form.apiKey"
            type="password"
            show-password
            autocomplete="new-password"
            :placeholder="apiKeySet ? apiKeyMasked : '未设置'"
          />
        </el-form-item>
        <el-form-item label="思考强度">
          <el-select
            v-model="form.reasoningEffort"
            filterable
            allow-create
            default-first-option
            placeholder="low"
            style="width: 100%"
          >
            <el-option v-for="item in reasoningOptions" :key="item" :label="item" :value="item" />
          </el-select>
        </el-form-item>
        <el-form-item label="温度">
          <el-input-number v-model="form.temperature" :min="0" :max="2" :step="0.1" />
        </el-form-item>
        <el-form-item label="写入方式">
          <el-radio-group v-model="form.writeMode">
            <el-radio-button value="confirm">确认后写入</el-radio-button>
            <el-radio-button value="auto">自动写入</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item>
          <template #label>
            <span class="label-with-tip">
              最大输出 token
              <el-tooltip
                content="模型单次回复最多生成的 Token 数量（非上下文窗口大小）。达到上限时生成会被截断。"
                placement="top"
              >
                <Icon icon="ant-design:question-circle-outlined" class="tip-icon" />
              </el-tooltip>
            </span>
          </template>
          <el-input-number v-model="form.maxTokens" :min="256" :max="128000" :step="1024" />
        </el-form-item>
        <el-form-item>
          <template #label>
            <span class="label-with-tip">
              上下文长度
              <el-tooltip
                content="输入给模型的提示词与历史消息总 Token 上限（默认 200000）。当消息长度超过此上限的 80% 时会自动触发摘要压缩。"
                placement="top"
              >
                <Icon icon="ant-design:question-circle-outlined" class="tip-icon" />
              </el-tooltip>
            </span>
          </template>
          <el-input-number v-model="form.maxPromptTokens" :min="1024" :max="1048576" :step="4096" />
        </el-form-item>
        <el-form-item label="最大轮次">
          <el-input-number v-model="form.maxTurns" :min="1" :max="32" />
        </el-form-item>
        <el-form-item label="单次超时(秒)">
          <el-input-number v-model="form.timeoutSeconds" :min="10" :max="300" />
        </el-form-item>
        <el-form-item label="整次超时(秒)">
          <el-input-number v-model="form.runTimeoutSeconds" :min="30" :max="1800" />
        </el-form-item>
        <el-form-item v-if="canAllowPrivateLlm" label="允许 HTTP/私网">
          <el-switch v-model="form.allowPrivateLlm" />
        </el-form-item>
        <el-form-item v-if="apiKeySet">
          <el-button @click="clearKey">清除密钥</el-button>
        </el-form-item>
      </el-form>
    </Dialog>
  </div>
</template>

<script>
import { getAgentSettings, saveAgentSettings } from '../api.js'

export default {
  name: 'AgentModelDialog',
  data() {
    return {
      apiKeySet: false,
      apiKeyMasked: '',
      canAllowPrivateLlm: false,
      reasoningOptions: ['low', 'medium', 'high', 'xhigh'],
      presets: [
        {
          id: 'deepseek',
          label: 'DeepSeek',
          baseUrl: 'https://api.deepseek.com',
          models: ['deepseek-flash', 'deepseek-v4-pro']
        },
        {
          id: 'qwen',
          label: '千问 百炼',
          baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
          models: ['qwen3.8-max', 'qwen3.7-plus', 'qwen3.8-flash', 'qwen3.7-flash']
        },
        {
          id: 'glm',
          label: '智谱 GLM',
          baseUrl: 'https://open.bigmodel.cn/api/paas/v4',
          models: ['glm-5.2', 'glm-5', 'glm-4.6']
        },
        {
          id: 'kimi',
          label: 'Kimi 月之暗面',
          baseUrl: 'https://api.moonshot.cn/v1',
          models: ['kimi-k2.5', 'kimi-k2.7-code', 'kimi-k3']
        }
      ],
      placeholder: { baseUrl: '', model: '' },
      form: {
        baseUrl: '',
        model: '',
        apiKey: '',
        reasoningEffort: 'low',
        temperature: 0.2,
        writeMode: 'confirm',
        maxTokens: 16384,
        maxPromptTokens: 200000,
        maxTurns: 16,
        timeoutSeconds: 90,
        runTimeoutSeconds: 600,
        allowPrivateLlm: false,
        apiKeyClear: false
      }
    }
  },
  computed: {
    activePreset() {
      return this.matchPreset(this.form.baseUrl)
    },
    modelOptions() {
      const list = (this.activePreset && this.activePreset.models) || []
      const cur = (this.form.model || '').trim()
      if (cur && list.indexOf(cur) < 0) return [cur].concat(list)
      return list
    }
  },
  methods: {
    normUrl(u) {
      return String(u || '')
        .trim()
        .replace(/\/+$/, '')
    },
    matchPreset(baseUrl) {
      const u = this.normUrl(baseUrl)
      if (!u) return null
      const aliases = {
        'https://api.deepseek.com/v1': 'https://api.deepseek.com'
      }
      const key = aliases[u] || u
      return (
        this.presets.find(
          (p) => this.normUrl(p.baseUrl) === key || this.normUrl(p.baseUrl) === u
        ) || null
      )
    },
    onBaseUrlChange(url) {
      const p = this.matchPreset(url)
      if (p && p.models && p.models.length && p.models.indexOf(this.form.model) < 0) {
        this.form.model = p.models[0]
      }
    },
    fillDefaults(r) {
      const src = r || {}
      this.placeholder = {
        baseUrl: src.defaultBaseUrl || '',
        model: src.defaultModel || ''
      }
      this.form.baseUrl = src.baseUrl || this.placeholder.baseUrl
      this.form.model = src.model || this.placeholder.model
      this.form.reasoningEffort = src.reasoningEffort || 'low'
    },
    open(cfg) {
      getAgentSettings()
        .then((resp) => {
          const r = (resp && resp.result) || {}
          this.apiKeySet = !!r.apiKeySet
          this.apiKeyMasked = r.apiKeyMasked || ''
          this.fillDefaults(r)
          this.form.apiKey = ''
          this.form.temperature = r.temperature == null ? 0.2 : r.temperature
          this.form.writeMode = r.writeMode || 'confirm'
          this.form.maxTokens = r.maxTokens || 16384
          this.form.maxPromptTokens = r.maxPromptTokens || 200000
          this.form.maxTurns = r.maxTurns || 16
          this.form.timeoutSeconds = r.timeoutSeconds || 90
          this.form.runTimeoutSeconds = r.runTimeoutSeconds || 600
          this.canAllowPrivateLlm = !!r.canAllowPrivateLlm
          this.form.allowPrivateLlm = this.canAllowPrivateLlm ? !!r.allowPrivateLlm : false
          this.form.apiKeyClear = false
          this.$refs.dlg.open({ title: (cfg && cfg.title) || '配置模型' })
        })
        .catch(() => {
          this.fillDefaults({})
          this.$refs.dlg.open({ title: (cfg && cfg.title) || '配置模型' })
        })
    },
    close() {
      this.$refs.dlg.close()
    },
    clearKey() {
      this.form.apiKeyClear = true
      this.form.apiKey = ''
    },
    onConfirm() {
      this.$refs.formRef.validate((valid) => {
        if (!valid) return
        const body = {
          baseUrl: this.form.baseUrl,
          model: this.form.model,
          reasoningEffort: this.form.reasoningEffort,
          temperature: this.form.temperature,
          writeMode: this.form.writeMode,
          maxTokens: this.form.maxTokens,
          maxPromptTokens: this.form.maxPromptTokens,
          maxTurns: this.form.maxTurns,
          timeoutSeconds: this.form.timeoutSeconds,
          runTimeoutSeconds: this.form.runTimeoutSeconds,
          allowPrivateLlm: this.canAllowPrivateLlm ? this.form.allowPrivateLlm : false
        }
        if (this.form.apiKeyClear) body.apiKeyClear = true
        else if (this.form.apiKey) body.apiKey = this.form.apiKey
        saveAgentSettings(body).then((resp) => {
          this.$emit('confirm', resp && resp.result)
          this.close()
        })
      })
    }
  }
}
</script>

<style scoped>
.label-with-tip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  white-space: nowrap;
}
.tip-icon {
  color: var(--el-text-color-secondary);
  cursor: pointer;
  font-size: 14px;
}
.tip-icon:hover {
  color: var(--el-color-primary);
}
</style>
