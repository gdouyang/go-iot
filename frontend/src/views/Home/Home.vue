<script setup lang="ts">
import { useI18n } from '@/hooks/web/useI18n'
import { ref } from 'vue'
import { useAppStore } from '@/store/modules/app'
defineOptions({
  name: 'Menu2'
})

const { t } = useI18n()

const appStore = useAppStore()
const text = ref(t('login.welcome') + appStore.getTitle)
</script>

<template>
  <div><el-empty :description="text" /></div>
</template>
