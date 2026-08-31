<script setup lang="ts">
import { ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus'
import { useI18n } from '@/hooks/web/useI18n'
import { useDesign } from '@/hooks/web/useDesign'
import LockDialog from './components/LockDialog.vue'
import { ref, computed } from 'vue'
import LockPage from './components/LockPage.vue'
import { useLockStore } from '@/store/modules/lock'
import { useUserStore } from '@/store/modules/user'
import { useRouter } from 'vue-router'

const { push } = useRouter()

const userStore = useUserStore()

const lockStore = useLockStore()

const getIsLock = computed(() => lockStore.getLockInfo?.isLock ?? false)

const { getPrefixCls } = useDesign()

const prefixCls = getPrefixCls('user-info')

const { t } = useI18n()

const dialogVisible = ref<boolean>(false)

/** 关闭下拉时先移除焦点，避免 aria-hidden 与焦点冲突告警 */
const clearDropdownFocus = () => {
  const el = document.activeElement as HTMLElement | null
  if (el && typeof el.blur === 'function') {
    el.blur()
  }
}

const handleCommand = (command: string | number | object) => {
  clearDropdownFocus()
  switch (command) {
    case 'personal':
      push('/personal/personal-center')
      break
    case 'lock':
      dialogVisible.value = true
      break
    case 'logout':
      userStore.logoutConfirm()
      break
  }
}
</script>

<template>
  <ElDropdown class="custom-hover" :class="prefixCls" trigger="click" @command="handleCommand">
    <div class="flex items-center">
      <span class="text-14px text-[var(--top-header-text-color)] cursor-pointer">{{
        userStore.getUserInfo?.username
      }}</span>
    </div>
    <template #dropdown>
      <ElDropdownMenu>
        <ElDropdownItem command="personal">
          {{ t('router.personalCenter') }}
        </ElDropdownItem>
        <ElDropdownItem command="lock" divided>
          {{ t('lock.lockScreen') }}
        </ElDropdownItem>
        <ElDropdownItem command="logout">
          {{ t('common.loginOut') }}
        </ElDropdownItem>
      </ElDropdownMenu>
    </template>
  </ElDropdown>

  <LockDialog v-if="dialogVisible" v-model="dialogVisible" />
  <teleport to="body">
    <transition name="fade-bottom" mode="out-in">
      <LockPage v-if="getIsLock" />
    </transition>
  </teleport>
</template>

<style scoped lang="less">
.fade-bottom-enter-active,
.fade-bottom-leave-active {
  transition:
    opacity 0.25s,
    transform 0.3s;
}

.fade-bottom-enter-from {
  opacity: 0;
  transform: translateY(-10%);
}

.fade-bottom-leave-to {
  opacity: 0;
  transform: translateY(10%);
}
</style>
