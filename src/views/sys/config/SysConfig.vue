<template>
  <ContentWrap title="系统配置">
    <el-form ref="form" :model="mdl" label-width="120px" style="width: 500px">
      <el-form-item
        label="系统名称"
        prop="title"
        :rules="[{ required: true, message: '请输入系统名称' }]"
      >
        <el-input v-model="mdl.title" :maxlength="8" />
      </el-form-item>

      <el-form-item label="简介">
        <el-input v-model="mdl.desc" :maxlength="30" />
      </el-form-item>

      <el-form-item label="默认位置">
        <el-input v-model="defaultLocation" :disabled="true">
          <el-button
            slot="addonAfter"
            @click="openLocation"
            type="link"
            icon="setting"
            style="width: auto; height: auto"
          />
        </el-input>
      </el-form-item>

      <el-form-item label="接入IP">
        <el-input v-model="mdl.accessIp" :max-length="128" />
      </el-form-item>

      <el-form-item>
        <div>
          <el-image style="width: 100px; height: 100px" :src="img" />
        </div>
        <el-upload
          name="file"
          :multiple="false"
          accept=".jpg,.png"
          :show-file-list="false"
          :with-credentials="true"
          :before-upload="beforeUpload"
        >
          <BaseButton>
            <Icon icon="el:Upload" />
            选择文件
          </BaseButton>
        </el-upload>
      </el-form-item>

      <el-form-item>
        <el-button type="primary" @click="saveBasic">更新</el-button>
      </el-form-item>
    </el-form>
    <!-- <LocationConfig ref="LocationConfig" @success="selectLocation" /> -->
  </ContentWrap>
</template>

<script lang="jsx">
import _ from 'lodash-es'
import { useAppStore } from '@/store/modules/app'
import { saveSysConfig, uploadFile } from '../api.js'
// import LocationConfig from '@/components/tools/LocationConfig.vue'

const appStore = useAppStore()
export default {
  name: 'SysConfig',
  components: {},
  mixins: [],
  data() {
    return {
      mdl: {
        title: undefined,
        desc: undefined,
        img: undefined,
        defaultLocation: {
          longitude: 0,
          latitude: 0
        },
        accessIp: null
      },
      mode: 'inline'
    }
  },
  computed: {
    img() {
      // if (_.startsWith(this.mdl.img, 'api')) {
      //   return this.mdl.img
      // }
      // return 'api/' + this.mdl.img
      return this.mdl.img
    },
    defaultLocation() {
      const l = this.mdl.defaultLocation || {}
      return _.toString(l.longitude) + ',' + _.toString(l.latitude)
    }
  },
  mounted() {
    this.init()
  },
  methods: {
    init() {
      const data = _.cloneDeep(appStore.getSysConfig)
      if (!data.img) {
        data.img = undefined
      }
      if (!data.defaultLocation) {
        data.defaultLocation = { longitude: 0, latitude: 0 }
      }
      if (!data.accessIp) {
        data.accessIp = '127.0.0.1'
      }
      this.mdl = data
    },
    beforeUpload(file) {
      uploadFile(file).then((resp) => {
        if (resp.success) {
          this.mdl.img = resp.result
          this.$message.success('上传成功')
        }
      })
      return false
    },
    saveBasic() {
      this.$refs.form.validate((valid) => {
        if (valid) {
          // if (_.startsWith(this.mdl.img, 'api/')) {
          //   this.mdl.img = this.mdl.img.replace('api/', '')
          // }
          saveSysConfig({ id: 'sysconfig', config: this.mdl }).then((resp) => {
            if (resp.success) {
              this.$message.success('操作成功')
              appStore.setSysConfig(this.mdl)
            }
          })
        }
      })
    },
    openLocation() {
      this.$refs.LocationConfig.open(this.mdl.defaultLocation || {})
    },
    selectLocation(value) {
      this.mdl.defaultLocation = value
      this.$refs.LocationConfig.close()
    }
  }
}
</script>

<style lang="less" scoped>
.account-settings-info-main {
  width: 100%;
  display: flex;
  height: 100%;
  overflow: auto;

  &.mobile {
    display: block;

    .account-settings-info-left {
      border-right: unset;
      border-bottom: 1px solid #e8e8e8;
      width: 100%;
      height: 50px;
      overflow-x: auto;
      overflow-y: scroll;
    }
    .account-settings-info-right {
      padding: 20px 40px;
    }
  }

  .account-settings-info-left {
    border-right: 1px solid #e8e8e8;
    width: 224px;
  }

  .account-settings-info-right {
    flex: 1 1;
    padding: 8px 40px;

    .account-settings-info-title {
      color: rgba(0, 0, 0, 0.85);
      font-size: 20px;
      font-weight: 500;
      line-height: 28px;
      margin-bottom: 12px;
    }
    .account-settings-info-view {
      padding-top: 12px;
    }
  }
}
</style>
