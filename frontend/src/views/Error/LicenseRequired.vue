<template>
  <div class="license-required-container">
    <el-result
      icon="warning"
      title="系统授权已受限"
      sub-title="当前系统尚未导入有效 License 授权文件或授权已过期。"
    >
      <template #extra>
        <div class="tip-box">
          <p class="tip-text">
            您当前登录的账号<strong>无权进行 License 授权管理</strong>。请联系系统管理员登录并导入
            License 授权文件。
          </p>
          <div v-if="machineFingerprint" class="fingerprint-box">
            <span class="label">服务器机器码：</span>
            <code>{{ machineFingerprint }}</code>
            <el-button type="primary" link size="small" @click="copyCode">复制</el-button>
          </div>
        </div>

        <div class="actions">
          <el-button type="primary" :loading="checking" @click="handleRecheck">
            重新检查授权
          </el-button>
          <el-button @click="handleLogout"> 退出登录 </el-button>
        </div>
      </template>
    </el-result>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useUserStore } from '@/store/modules/user'
import { refreshLicenseStatus } from '@/permission'
import { getAnonLicenseStatus } from '@/views/sys/api'
import { ElMessage } from 'element-plus'

const router = useRouter()
const userStore = useUserStore()
const checking = ref(false)
const machineFingerprint = ref('')

const fetchFingerprint = async () => {
  try {
    const res = await getAnonLicenseStatus()
    if (res?.machineFingerprint) {
      machineFingerprint.value = res.machineFingerprint
    }
  } catch (_) {}
}

const copyCode = () => {
  if (!machineFingerprint.value) return
  navigator.clipboard.writeText(machineFingerprint.value).then(() => {
    ElMessage.success('机器指纹码已复制到剪贴板，可发送给管理员')
  })
}

const handleRecheck = async () => {
  checking.value = true
  try {
    const res = await refreshLicenseStatus()
    if (res?.isLicensed) {
      ElMessage.success('系统授权已生效，正在进入系统...')
      router.replace('/')
    } else {
      ElMessage.warning('系统当前仍未获得有效授权，请联系管理员上传证书')
    }
  } catch (err: any) {
    ElMessage.error(err.message || '检测失败，请稍后重试')
  } finally {
    checking.value = false
  }
}

const handleLogout = () => {
  userStore.logout()
}

onMounted(() => {
  fetchFingerprint()
})
</script>

<style scoped>
.license-required-container {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 80vh;
  padding: 40px 20px;
}

.tip-box {
  background: #fdf6ec;
  border: 1px solid #faecd8;
  border-radius: 8px;
  padding: 16px 24px;
  max-width: 540px;
  margin: 0 auto 24px auto;
  text-align: left;
}

.tip-text {
  font-size: 14px;
  color: #e6a23c;
  line-height: 1.6;
  margin: 0 0 12px 0;
}

.fingerprint-box {
  background: #fff;
  border: 1px dashed #e4e7ed;
  border-radius: 4px;
  padding: 8px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: 13px;
}

.fingerprint-box .label {
  color: #909399;
}

.fingerprint-box code {
  font-weight: bold;
  color: #303133;
}

.actions {
  display: flex;
  justify-content: center;
  gap: 16px;
}
</style>
