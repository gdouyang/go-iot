<template>
  <el-form-item label="文件类型">
    <el-select v-model="data.fileType">
      <el-option value="url">URL(链接)</el-option>
      <el-option value="base64">Base64(Base64编码)</el-option>
      <!-- <el-option value="binary">Binary(二进制)</el-option> --> </el-select
    >,
  </el-form-item>
</template>

<script lang="jsx">
export default {
  name: 'FileItem',
  components: {},
  props: {
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {}
  },
  created() {},
  mounted() {},
  methods: {}
}
</script>
