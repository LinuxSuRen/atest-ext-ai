<template>
  <div class="chat-input">
    <div class="input-options">
      <div class="spacer"></div>
      <el-button class="configure-btn" type="primary" plain @click="emit('open-settings')">
        <el-icon><Setting /></el-icon>
        {{ t('ai.button.configure') }}
      </el-button>
    </div>
    <div class="input-controls">
      <el-input
        v-model="prompt"
        type="textarea"
        :rows="3"
        :placeholder="t('ai.input.placeholder')"
        :disabled="props.loading"
        @keydown.enter.ctrl="handleSubmit"
        @keydown.enter.meta="handleSubmit"
      />
      <el-button
        type="primary"
        :loading="props.loading"
        :disabled="!prompt.trim()"
        @click="handleSubmit"
      >
        <el-icon v-if="!props.loading"><Promotion /></el-icon>
        {{ props.loading ? t('ai.message.generating') : t('ai.button.generate') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, inject } from 'vue'
import { Promotion, Setting } from '@element-plus/icons-vue'
import type { AppContext } from '../types'

interface Props {
  loading: boolean
  includeExplanation: boolean
}
const props = defineProps<Props>()

interface Emits {
  (e: 'submit', prompt: string, options: { includeExplanation: boolean }): void
  (e: 'open-settings'): void
}
const emit = defineEmits<Emits>()

// Inject context
const context = inject<AppContext>('appContext')!
const { t } = context.i18n

// Input state
const prompt = ref('')

function handleSubmit() {
  if (!prompt.value.trim() || props.loading) return

  emit('submit', prompt.value, {
    includeExplanation: props.includeExplanation
  })

  // Clear input after submit
  prompt.value = ''
}
</script>

<style scoped>
.chat-input {
  padding: 20px 40px 24px;
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
  box-shadow: 0 -4px 12px var(--el-box-shadow-lighter);
}

.input-options {
  margin-bottom: 12px;
  padding-left: 4px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 16px;
}

.spacer {
  flex: 1;
}

.configure-btn {
  display: flex;
  align-items: center;
  gap: 6px;
  height: 44px;
  padding: 0 24px;
  border-radius: 12px;
  font-weight: 500;
}

.input-controls {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.input-controls :deep(.el-textarea__inner) {
  border-radius: 12px;
  border: 2px solid var(--el-border-color);
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.6;
  resize: none;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px var(--el-box-shadow-lighter);
}

.input-controls :deep(.el-textarea__inner:focus) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px var(--el-color-primary-light-9);
}

.input-controls :deep(.el-textarea__inner::placeholder) {
  color: var(--el-text-color-placeholder);
}

.input-controls .el-button {
  height: 48px;
  padding: 0 28px;
  border-radius: 12px;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  background: var(--el-color-primary);
  border: none;
  box-shadow: 0 4px 12px var(--el-box-shadow);
  transition: all 0.3s ease;
}

.input-controls .el-button:hover:not(:disabled) {
  transform: translateY(-2px);
  background: var(--el-color-primary-light-3);
  box-shadow: 0 6px 16px var(--el-box-shadow-dark);
}

.input-controls .el-button:active:not(:disabled) {
  transform: translateY(0);
}

.input-controls .el-button:disabled {
  background: var(--el-fill-color);
  box-shadow: none;
}

.input-controls .el-button.is-loading {
  background: var(--el-color-primary);
  opacity: 0.8;
}
</style>
