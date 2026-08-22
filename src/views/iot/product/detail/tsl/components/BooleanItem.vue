<template>
  <div>
    <el-form-item label="布尔值" style="height: 69px">
      <el-row>
        <el-col :span="11">
          <el-input v-model="data.trueValue" placeholder="trueValue" />
        </el-col>
        <el-col :span="2" :push="1"> ~ </el-col>
        <el-col :span="11">
          <el-input v-model="data.trueText" placeholder="true名称" />
        </el-col>
      </el-row>
      <el-row>
        <el-col :span="11">
          <el-input v-model="data.falseValue" placeholder="falseValue" />
        </el-col>
        <el-col :span="2" :push="1"> ~ </el-col>
        <el-col :span="11">
          <el-input v-model="data.falseText" placeholder="false名称" />
        </el-col>
      </el-row>
    </el-form-item>
  </div>
</template>

<script lang="jsx">
export default {
  name: 'BooleanItem',
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
