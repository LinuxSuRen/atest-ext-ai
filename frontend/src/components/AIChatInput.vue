<template>
  <div class="chat-input">
    <div class="input-box">
      <el-input
        v-model="prompt"
        class="prompt-input"
        type="textarea"
        :rows="3"
        :placeholder="t('ai.input.placeholder')"
        :disabled="props.loading"
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
        @keydown.enter.ctrl="handleSubmit"
        @keydown.enter.meta="handleSubmit"
      />
      <div class="action-buttons">
        <el-tooltip :content="configureTooltip" placement="left">
          <el-button
            class="icon-btn configure-btn"
            type="primary"
            plain
            circle
            :aria-label="configureTooltip"
            @click="emit('open-settings')"
          >
            <el-icon><Setting /></el-icon>
          </el-button>
        </el-tooltip>
        <el-tooltip :content="generateTooltip" placement="left">
          <el-button
            class="icon-btn generate-btn"
            type="primary"
            circle
            :aria-label="generateTooltip"
            :loading="props.loading"
            :disabled="!prompt.trim()"
            @click="handleSubmit"
          >
            <el-icon v-if="!props.loading"><Promotion /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, inject, computed } from 'vue'
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

const configureTooltip = computed(() => t('ai.tooltip.configure'))
const generateTooltip = computed(() => (props.loading ? t('ai.message.generating') : t('ai.tooltip.generate')))

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

.input-box {
  position: relative;
}

.prompt-input {
  display: block;
}

.input-box :deep(.el-textarea__inner) {
  border-radius: 12px;
  border: 2px solid var(--el-border-color);
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.6;
  resize: none;
  transition: all 0.3s ease;
  box-shadow: 0 2px 8px var(--el-box-shadow-lighter);
  min-height: 124px;
  padding-right: 96px;
}

.input-box :deep(.el-textarea__inner:focus) {
  border-color: var(--el-color-primary);
  box-shadow: 0 0 0 3px var(--el-color-primary-light-9);
}

.input-box :deep(.el-textarea__inner::placeholder) {
  color: var(--el-text-color-placeholder);
}

.action-buttons {
  position: absolute;
  top: 16px;
  right: 20px;
  bottom: 16px;
  display: flex;
  flex-direction: column;
  justify-content: space-between;
  align-items: center;
  gap: 12px;
}

.icon-btn {
  box-sizing: border-box;
  display: flex;
  justify-content: center;
  align-items: center;
  width: 48px;
  height: 48px;
  transition: all 0.3s ease;
  backdrop-filter: blur(4px);
}

.configure-btn {
  border: 2px solid var(--el-color-primary-light-7);
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary-dark-2);
}

.configure-btn:hover {
  background: var(--el-color-primary-light-8);
  border-color: var(--el-color-primary-light-6);
}

.generate-btn {
  background: var(--el-color-primary);
  border: none;
  box-shadow: 0 4px 12px var(--el-box-shadow);
  color: var(--el-color-white);
  padding: 0;
}

.generate-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  background: var(--el-color-primary-light-3);
  box-shadow: 0 6px 16px var(--el-box-shadow-dark);
}

.generate-btn:active:not(:disabled) {
  transform: translateY(0);
}

.generate-btn:disabled {
  background: var(--el-fill-color);
  box-shadow: none;
  color: var(--el-text-color-placeholder);
}

.generate-btn.is-loading {
  background: var(--el-color-primary);
  opacity: 0.8;
}
</style>
