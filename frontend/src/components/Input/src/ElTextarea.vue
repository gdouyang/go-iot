<script setup lang="ts">
import { ElInput } from 'element-plus'
</script>

<template>
  <ElInput v-bind="$attrs" type="textarea" show-word-limit />
</template>
