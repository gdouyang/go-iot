<template>
  <el-form-item label="密码长度">
    <el-input-number
      v-model="data.max"
      :precision="0"
      :min="0"
      :step="1"
      controls-position="right"
      style="width: 100%"
    />
  </el-form-item>
</template>

<script lang="jsx">
export default {
  name: 'PasswordItem',
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
