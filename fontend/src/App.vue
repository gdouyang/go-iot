<template>
  <BaseHeader />
  <div style="padding: 10px">
    <!-- <BaseSide /> -->
    <div>
      <!-- <img alt="Vue logo" class="element-plus-logo" src="./assets/logo.png" />
      <HelloWorld msg="Hello Vue 3.0 + Element Plus + Vite" /> -->
      <router-view></router-view>
    </div>
  </div>
</template>

<style>
#app {
  text-align: center;
  color: var(--el-text-color-primary);
}

.element-plus-logo {
  width: 50%;
}
</style>
