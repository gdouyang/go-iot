<script setup lang="ts">
import router from '@/router'
// import { useTagsViewStore } from '@/store/modules/tagsView'
import { useAppStore } from '@/store/modules/app'
import { Footer } from '@/components/Footer'
import { computed } from 'vue'

const appStore = useAppStore()

const footer = computed(() => appStore.getFooter)

// const tagsViewStore = useTagsViewStore()

const keepAlive = computed((): Boolean => {
  // const caches = tagsViewStore.getCachedViews
  // if (!) {
  //   const routerName = router.currentRoute.value.name?.toString()
  //   if (routerName && !caches.includes(routerName)) {
  //     caches.push(routerName)
  //   }
  // }
  return !(router.currentRoute.value.meta.noCache || false)
})
</script>

<template>
  <section
    :class="[
      'flex-1 p-[var(--app-content-padding)] w-[calc(100%-var(--app-content-padding)-var(--app-content-padding))] bg-[var(--app-content-bg-color)] dark:bg-[var(--el-bg-color)]'
    ]"
  >
    <!-- <keep-alive>
      <router-view v-if="keepAlive"> </router-view>
    </keep-alive>
    <router-view v-if="!keepAlive"> </router-view> -->
    <router-view v-slot="{ Component }">
      <!-- <keep-alive> -->
      <component :is="Component" />
      <!-- </keep-alive> -->
    </router-view>
  </section>
  <Footer v-if="footer" />
</template>
