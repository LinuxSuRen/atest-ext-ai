<template>
  <div class="ai-chat-header">
    <div class="header-left">
      <h2>{{ t('ai.title') }}</h2>
      <span class="subtitle">{{ t('ai.subtitle') }}</span>
      <span class="provider-label">
        {{ providerLabelText }}
        <span class="status-indicator" :class="props.status">
          <span class="status-dot" />
          {{ statusText }}
        </span>
      </span>
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

const providerLabel = computed(() => {
  const key = props.provider === 'local' ? 'ollama' : props.provider
  const translationKey = `ai.provider.${key}.name`
  const translated = t(translationKey)
  return translated === translationKey ? key : translated
})

const providerLabelText = computed(() => `${t('ai.providerLabel')}: ${providerLabel.value}`)
const statusText = computed(() => t(`ai.status.${props.status}`))
</script>

<style scoped>
.ai-chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--atest-spacing-md) clamp(20px, 4vw, 28px);
  background: var(--atest-bg-surface);
  border-bottom: 1px solid var(--atest-border-color);
}

.header-left {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.header-left h2 {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  color: var(--atest-text-primary);
}

.subtitle {
  font-size: 13px;
  color: var(--atest-text-secondary);
}

.provider-label {
  display: block;
  font-size: 12px;
  color: var(--atest-text-regular);
  display: flex;
  align-items: center;
  gap: 8px;
}

.status-indicator {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 3px 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
  background: var(--atest-bg-elevated);
  color: var(--atest-text-regular);
  border: 1px solid var(--atest-border-color);
}

.status-indicator.connected {
  background: var(--el-color-success-light-9);
  color: var(--el-color-success-dark-2);
  border-color: var(--el-color-success-light-5);
}

.status-indicator.connecting {
  background: var(--el-color-warning-light-9);
  color: var(--el-color-warning-dark-2);
  border-color: var(--el-color-warning-light-5);
}

.status-indicator.disconnected {
  background: var(--el-color-danger-light-9);
  color: var(--el-color-danger-dark-2);
  border-color: var(--el-color-danger-light-5);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}

@media (max-width: 768px) {
  .ai-chat-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 8px;
    padding: 16px 20px;
  }

  .header-left h2 {
    font-size: 16px;
  }

  .subtitle {
    font-size: 12px;
  }
}

@media (max-width: 480px) {
  .ai-chat-header {
    padding: 14px 16px;
  }

  .status-indicator {
    font-size: 11px;
    padding: 2px 8px;
  }
}
</style>
