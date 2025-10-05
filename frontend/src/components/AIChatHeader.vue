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
      <el-button
        type="primary"
        size="small"
        @click="emit('open-settings')"
      >
        <el-icon><Setting /></el-icon>
        {{ t('ai.settings.title') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject } from 'vue'
import { Setting } from '@element-plus/icons-vue'
import type { AppContext } from '../types'

interface Props {
  provider: string
  status: 'connected' | 'disconnected' | 'connecting'
}
const props = defineProps<Props>()

interface Emits {
  (e: 'open-settings'): void
}
const emit = defineEmits<Emits>()

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
  background: #fff;
  border-bottom: 1px solid #e4e7ed;
}

.header-left h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: #303133;
}

.subtitle {
  margin-left: 12px;
  font-size: 13px;
  color: #909399;
}

.header-right {
  display: flex;
  gap: 12px;
  align-items: center;
}
</style>
