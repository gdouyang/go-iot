<template>
  <ContentWrap title="授权管理">
    <!-- 状态提示横幅 (仅在加载完成后根据状态展示) -->
    <template v-if="loaded">
      <el-alert
        v-if="!info.isLicensed"
        :title="
          info.status === 'expired'
            ? '系统 License 授权已过期，部分核心功能已受限，请尽快导入新证书！'
            : '系统当前处于未授权状态，请先获取服务器机器码并向厂商申请导入 License 授权证书！'
        "
        type="error"
        :closable="false"
        show-icon
        style="margin-bottom: 20px"
      />
      <el-alert
        v-else-if="info.status === 'expiring_soon'"
        title="系统 License 授权即将到期（15天内），请及时联系厂商续期证书以防业务中断！"
        type="warning"
        :closable="false"
        show-icon
        style="margin-bottom: 20px"
      />
    </template>

    <el-row :gutter="20">
      <!-- 左侧：授权详情卡片 -->
      <el-col :span="14">
        <el-card shadow="hover" header="当前授权状态" v-loading="loading">
          <el-descriptions :column="1" border size="large">
            <el-descriptions-item label="授权状态">
              <el-tag :type="getStatusTagType(info.status)" effect="dark">
                {{ info.statusText || '未授权' }}
              </el-tag>
            </el-descriptions-item>

            <el-descriptions-item label="客户名称">
              <span style="font-weight: bold; font-size: 15px">
                {{ info.customerName || '未授权客户' }}
              </span>
            </el-descriptions-item>

            <el-descriptions-item label="License ID">
              <code style="font-size: 12px">{{ info.licenseId || '-' }}</code>
            </el-descriptions-item>

            <el-descriptions-item label="设备接入配额">
              <div style="width: 100%">
                <div style="display: flex; justify-content: space-between; margin-bottom: 4px">
                  <span
                    >当前接入：<strong>{{ info.currentDevices || 0 }}</strong> 台</span
                  >
                  <span
                    >上限配额：<strong>{{
                      info.maxDevices > 0 ? info.maxDevices + ' 台' : '无限制'
                    }}</strong></span
                  >
                </div>
                <el-progress
                  v-if="info.maxDevices > 0"
                  :percentage="calcDevicePercentage"
                  :status="
                    calcDevicePercentage >= 90
                      ? 'exception'
                      : calcDevicePercentage >= 75
                        ? 'warning'
                        : 'success'
                  "
                />
              </div>
            </el-descriptions-item>

            <el-descriptions-item label="授权生效时间">
              {{ info.issuedAtFormatted || '-' }}
            </el-descriptions-item>

            <el-descriptions-item label="授权到期时间">
              <span
                :style="{
                  color: info.status === 'expired' ? '#f56c6c' : 'inherit',
                  fontWeight: 'bold'
                }"
              >
                {{ info.expiresAtFormatted || '永久有效' }}
              </span>
              <span
                v-if="remainingDaysText"
                style="margin-left: 10px; font-size: 12px; color: #909399"
              >
                ({{ remainingDaysText }})
              </span>
            </el-descriptions-item>
          </el-descriptions>

          <!-- 授权有效时的快捷跳转 -->
          <div
            v-if="info.isLicensed"
            style="margin-top: 20px; display: flex; gap: 12px; align-items: center"
          >
            <el-button type="primary" @click="goToHome">进入系统首页</el-button>
            <el-button type="success" @click="goToDevice">前往设备管理</el-button>
            <el-button @click="goToProduct">前往产品管理</el-button>
          </div>
        </el-card>
      </el-col>

      <!-- 右侧：机器码获取与证书导入 -->
      <el-col :span="10">
        <!-- 机器指纹卡片 -->
        <el-card shadow="hover" header="服务器机器指纹码" style="margin-bottom: 20px">
          <p style="font-size: 13px; color: #606266; margin-bottom: 12px">
            将当前服务器的机器指纹码复制并提供给厂商，用于签发绑定本服务器的专属 License 文件：
          </p>
          <div
            style="
              background: #f4f4f5;
              padding: 12px;
              border-radius: 6px;
              display: flex;
              align-items: center;
              justify-content: space-between;
            "
          >
            <code style="font-weight: bold; color: #303133; font-size: 14px">{{
              info.machineFingerprint || '获取中...'
            }}</code>
            <el-button type="primary" link @click="copyMachineCode">复制机器码</el-button>
          </div>
        </el-card>

        <!-- 导入 License 卡片 -->
        <el-card shadow="hover" header="导入 / 更新 License 证书">
          <el-upload
            drag
            action="#"
            :auto-upload="false"
            :show-file-list="false"
            accept=".lic"
            :on-change="handleFileChange"
          >
            <el-icon class="el-icon--upload" style="font-size: 48px; color: #409eff"
              ><upload-filled
            /></el-icon>
            <div class="el-upload__text">
              将 <em>.lic</em> 证书文件拖拽至此处，或 <em>点击选择文件</em>
            </div>
            <template #tip>
              <div class="el-upload__tip"
                >仅支持导入由发证工具签发的 .lic 格式文件，导入后立即生效。</div
              >
            </template>
          </el-upload>
        </el-card>
      </el-col>
    </el-row>
  </ContentWrap>
</template>

<script>
import { defineComponent, ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { getLicenseInfo, uploadLicense } from './api'
import { refreshLicenseStatus } from '@/permission'
import { ElMessage } from 'element-plus'
import { UploadFilled } from '@element-plus/icons-vue'

export default defineComponent({
  name: 'LicensePage',
  components: {
    UploadFilled
  },
  setup() {
    const router = useRouter()
    const loading = ref(true)
    const loaded = ref(false)

    const info = ref({
      status: 'unlicensed',
      statusText: '加载中...',
      customerName: '',
      licenseId: '',
      issuedAtFormatted: '',
      expiresAt: 0,
      expiresAtFormatted: '',
      maxDevices: 0,
      currentDevices: 0,
      fingerprint: '',
      machineFingerprint: '',
      isLicensed: false,
      requireRedirect: false
    })

    const fetchInfo = async () => {
      loading.value = true
      try {
        const res = await getLicenseInfo()
        if (res) {
          info.value = res
        }
      } catch (err) {
        ElMessage.error(err.message || '获取授权信息失败')
      } finally {
        loading.value = false
        loaded.value = true
      }
    }

    const calcDevicePercentage = computed(() => {
      if (!info.value.maxDevices || info.value.maxDevices <= 0) return 0
      const pct = Math.round((info.value.currentDevices / info.value.maxDevices) * 100)
      return pct > 100 ? 100 : pct
    })

    const remainingDaysText = computed(() => {
      if (!info.value.expiresAt || info.value.expiresAt <= 0) return ''
      const nowSec = Math.floor(Date.now() / 1000)
      const diffSec = info.value.expiresAt - nowSec
      if (diffSec <= 0) return '已过期'
      const days = Math.ceil(diffSec / 86400)
      return `剩余 ${days} 天`
    })

    const getStatusTagType = (status) => {
      switch (status) {
        case 'active':
          return 'success'
        case 'expiring_soon':
          return 'warning'
        case 'expired':
        case 'fingerprint_mismatch':
        case 'invalid_signature':
        case 'unlicensed':
        default:
          return 'danger'
      }
    }

    const copyMachineCode = () => {
      if (!info.value.machineFingerprint) return
      navigator.clipboard.writeText(info.value.machineFingerprint).then(() => {
        ElMessage.success('机器指纹码已复制到剪贴板')
      })
    }

    const handleFileChange = async (uploadFile) => {
      if (!uploadFile || !uploadFile.raw) return
      try {
        await uploadLicense(uploadFile.raw)
        await refreshLicenseStatus()
        ElMessage.success('License 证书导入并激活成功！')
        await fetchInfo()
      } catch (err) {
        ElMessage.error(err.message || '导入 License 失败')
      }
    }

    const goToHome = () => {
      router.push('/')
    }

    const goToDevice = () => {
      router.push('/device/device-list')
    }

    const goToProduct = () => {
      router.push('/product/product-list')
    }

    onMounted(() => {
      fetchInfo()
    })

    return {
      loading,
      loaded,
      info,
      calcDevicePercentage,
      remainingDaysText,
      getStatusTagType,
      copyMachineCode,
      handleFileChange,
      goToHome,
      goToDevice,
      goToProduct
    }
  }
})
</script>
