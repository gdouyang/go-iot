<template>
  <div>
    <el-form-item label="精度">
      <el-input-number
        v-model="data.scale"
        :precision="0"
        :min="0"
        :max="10"
        :step="1"
        controls-position="right"
        placeholder="小数点位数"
        style="width: 100%"
      />
    </el-form-item>
    <el-form-item label="单位">
      <el-input v-model="data.unit" :maxlength="32" style="width: 100%" />
    </el-form-item>
  </div>
</template>

<script lang="jsx">
export default {
  name: 'NumberItem',
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
