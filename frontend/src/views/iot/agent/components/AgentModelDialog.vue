<template>
  <div>
    <Dialog ref="dlg" :show-ok="true" :ok-text="'保存'" @confirm="onConfirm" maxHeight="auto">
      <el-form ref="formRef" :model="form" label-width="130px">
        <el-form-item
          label="模型 URL"
          prop="baseUrl"
          :rules="[
            { required: true, whitespace: true, message: '请输入模型 URL', trigger: 'blur' }
          ]"
        >
          <el-input v-model="form.baseUrl" :placeholder="placeholder.baseUrl" />
        </el-form-item>
        <el-form-item
          label="模型"
          prop="model"
          :rules="[{ required: true, whitespace: true, message: '请输入模型', trigger: 'blur' }]"
        >
          <el-input v-model="form.model" :placeholder="placeholder.model" />
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
        <el-form-item label="最大输出 token">
          <el-input-number v-model="form.maxTokens" :min="256" :max="128000" :step="1024" />
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
        <!-- 暂不开放：允许 HTTP/私网，默认 false
        <el-form-item label="允许 HTTP/私网">
          <el-switch v-model="form.allowPrivateLlm" />
        </el-form-item>
        -->
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
      reasoningOptions: ['low', 'medium', 'high', 'xhigh'],
      placeholder: { baseUrl: '', model: '' },
      form: {
        baseUrl: '',
        model: '',
        apiKey: '',
        reasoningEffort: 'low',
        temperature: 0.2,
        writeMode: 'confirm',
        maxTokens: 16384,
        maxTurns: 16,
        timeoutSeconds: 90,
        runTimeoutSeconds: 600,
        allowPrivateLlm: false,
        apiKeyClear: false
      }
    }
  },
  methods: {
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
          this.form.maxTurns = r.maxTurns || 16
          this.form.timeoutSeconds = r.timeoutSeconds || 90
          this.form.runTimeoutSeconds = r.runTimeoutSeconds || 600
          this.form.allowPrivateLlm = false
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
          maxTurns: this.form.maxTurns,
          timeoutSeconds: this.form.timeoutSeconds,
          runTimeoutSeconds: this.form.runTimeoutSeconds,
          allowPrivateLlm: false
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
