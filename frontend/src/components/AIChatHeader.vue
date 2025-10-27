<template>
  <div class="ai-chat-header">
    <div class="header-left">
      <h2>{{ t('ai.title') }}</h2>
      <span class="subtitle">{{ t('ai.subtitle') }}</span>
    </div>
    <div class="header-right">
      <el-tag :type="statusType" size="small">
        {{ t(`ai.status.${props.status}`) }}
      </el-tag>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import type { AppContext } from '../types'

interface Props {
  provider: string
  status: 'connected' | 'disconnected' | 'connecting'
}
const props = defineProps<Props>()

// Inject context from parent
const context = inject<AppContext>('appContext')!
const { t } = context.i18n

// Status tag type
const statusType = computed(() => {
  switch (props.status) {
    case 'connected': return 'success'
    case 'connecting': return 'warning'
    default: return 'info'
  }
})
</script>

<style scoped>
.ai-chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 24px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color);
}

.header-left h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.subtitle {
  margin-left: 12px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}

.header-right {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
