<template>
  <el-form-item label="时间格式">
    <el-select
      filterable
      clearable
      style="width: 100%"
      v-model="data.format"
      placeholder="默认格式：String类型的UTC时间戳 (毫秒)"
    >
      <el-option
        v-for="item in dateSource"
        :key="item.value"
        :label="item.text"
        :value="item.value"
      />
    </el-select>
  </el-form-item>
</template>

<script lang="jsx">
export default {
  name: 'DateItem',
  components: {},
  props: {
    data: {
      type: Object,
      default: () => {}
    }
  },
  data() {
    return {
      dateSource: [
        {
          text: 'String类型的UTC时间戳 (毫秒)',
          value: 'string'
        },
        {
          text: 'yyyy-MM-dd',
          value: 'yyyy-MM-dd'
        },
        {
          text: 'yyyy-MM-dd HH:mm:ss',
          value: 'yyyy-MM-dd HH:mm:ss'
        },
        {
          text: 'yyyy-MM-dd HH:mm:ss.SSS',
          value: 'yyyy-MM-dd HH:mm:ss.SSS'
        }
      ]
    }
  },
  created() {},
  mounted() {},
  methods: {}
}
</script>
