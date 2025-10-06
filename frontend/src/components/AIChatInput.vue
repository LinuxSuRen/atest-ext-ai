<template>
  <div class="chat-input">
    <div class="input-options">
      <el-checkbox v-model="includeExplanation">
        {{ t('ai.option.includeExplanation') }}
      </el-checkbox>
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
import { Promotion } from '@element-plus/icons-vue'
import type { AppContext } from '../types'

interface Props {
  loading: boolean
}
const props = defineProps<Props>()

interface Emits {
  (e: 'submit', prompt: string, options: { includeExplanation: boolean }): void
}
const emit = defineEmits<Emits>()

// Inject context
const context = inject<AppContext>('appContext')!
const { t } = context.i18n

// Input state
const prompt = ref('')
const includeExplanation = ref(false)

function handleSubmit() {
  if (!prompt.value.trim() || props.loading) return

  emit('submit', prompt.value, {
    includeExplanation: includeExplanation.value
  })

  // Clear input after submit
  prompt.value = ''
}
</script>

<style scoped>
.chat-input {
  padding: 20px 40px 24px;
  background: #fff;
  border-top: 1px solid #e4e7ed;
  box-shadow: 0 -4px 12px rgba(0, 0, 0, 0.05);
}

.input-options {
  margin-bottom: 12px;
  padding-left: 4px;
}

.input-controls {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.input-controls :deep(.el-textarea__inner) {
  border-radius: 12px;
  border: 2px solid #e4e7ed;
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.6;
  resize: none;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}

.input-controls :deep(.el-textarea__inner:focus) {
  border-color: #667eea;
  box-shadow: 0 0 0 3px rgba(102, 126, 234, 0.1);
}

.input-controls :deep(.el-textarea__inner::placeholder) {
  color: #c0c4cc;
}

.input-controls .el-button {
  height: 48px;
  padding: 0 28px;
  border-radius: 24px;
  font-size: 14px;
  font-weight: 500;
  white-space: nowrap;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border: none;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.4);
  transition: all 0.3s ease;
}

.input-controls .el-button:hover:not(:disabled) {
  transform: translateY(-2px);
  box-shadow: 0 6px 16px rgba(102, 126, 234, 0.5);
}

.input-controls .el-button:active:not(:disabled) {
  transform: translateY(0);
}

.input-controls .el-button:disabled {
  background: #c0c4cc;
  box-shadow: none;
}

.input-controls .el-button.is-loading {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  opacity: 0.8;
}
</style>
