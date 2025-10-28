<template>
  <div :class="['chat-input', { 'is-disabled': props.disabled }]">
    <transition name="status-fade">
      <div v-if="showStatusBanner" class="status-banner" :class="props.status">
        <el-icon v-if="statusIcon" :class="['status-icon', { spin: props.status === 'connecting' }]">
          <component :is="statusIcon" />
        </el-icon>
        <span class="banner-text">{{ statusBanner }}</span>
        <el-button
          v-if="showConfigureLink"
          class="banner-action"
          link
          size="small"
          @click="emit('open-settings')"
        >
          {{ t('ai.button.configure') }}
        </el-button>
      </div>
    </transition>
    <div :class="['input-box', { 'has-banner': showStatusBanner }]">
      <el-input
        v-model="prompt"
        class="prompt-input"
        type="textarea"
        :rows="3"
        :placeholder="t('ai.input.placeholder')"
        :disabled="props.loading || props.disabled"
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
            :disabled="props.disabled || !prompt.trim()"
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
import { Promotion, Setting, WarningFilled, InfoFilled, Loading } from '@element-plus/icons-vue'
import type { AppContext } from '../types'

interface Props {
  loading: boolean
  includeExplanation: boolean
  provider: string
  status: 'connected' | 'disconnected' | 'connecting' | 'setup'
  disabled: boolean
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

const providerKey = computed(() => (props.provider === 'local' ? 'ollama' : props.provider))
const providerLabel = computed(() => {
  const translationKey = `ai.provider.${providerKey.value}.name`
  const translated = t(translationKey)
  return translated === translationKey ? providerKey.value : translated
})

const statusBanner = computed(() => {
  if (props.status === 'connected') return ''
  const map: Record<Props['status'], string> = {
    connected: '',
    connecting: 'ai.statusBanner.connecting',
    disconnected: 'ai.statusBanner.disconnected',
    setup: 'ai.statusBanner.setup'
  }
  const key = map[props.status]
  if (!key) return ''
  const message = t(key)
  if (!message || message === key) return ''
  return message.replace('{provider}', providerLabel.value)
})

const statusIcon = computed(() => {
  switch (props.status) {
    case 'connecting':
      return Loading
    case 'disconnected':
      return WarningFilled
    case 'setup':
      return InfoFilled
    default:
      return null
  }
})

const showStatusBanner = computed(() => Boolean(statusBanner.value))
const showConfigureLink = computed(() => props.status === 'disconnected' || props.status === 'setup')

function handleSubmit() {
  if (!prompt.value.trim() || props.loading || props.disabled) return

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
  background: var(--atest-bg-surface);
  border-top: 1px solid var(--atest-border-color);
  box-shadow: var(--atest-shadow-sm);
  border-radius: var(--atest-radius-md);
  display: flex;
  flex-direction: column;
  gap: var(--atest-spacing-sm);
}

.input-box {
  position: relative;
}

.input-box.has-banner {
  margin-top: var(--atest-spacing-sm);
}

.prompt-input {
  display: block;
}

.input-box :deep(.el-textarea__inner) {
  border-radius: 12px;
  border: 2px solid var(--atest-border-color);
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.6;
  resize: none;
  transition: var(--atest-transition-base);
  box-shadow: 0 2px 8px var(--el-box-shadow-lighter);
  min-height: 124px;
  padding-right: 96px;
}

.input-box :deep(.el-textarea__inner:focus) {
  border-color: var(--atest-color-accent);
  box-shadow: 0 0 0 3px var(--atest-color-accent-soft);
}

.input-box :deep(.el-textarea__inner::placeholder) {
  color: var(--atest-text-placeholder);
}

.action-buttons {
  position: absolute;
  top: 16px;
  right: 20px;
  display: flex;
  flex-direction: column;
  justify-content: flex-start;
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
  transition: var(--atest-transition-base);
  backdrop-filter: blur(8px);
}

.configure-btn {
  border: none;
  background: var(--atest-color-accent-soft);
  color: var(--atest-color-accent);
}

.configure-btn:hover {
  background: var(--el-color-primary-light-7);
}

.generate-btn {
  background: var(--atest-color-accent);
  border: none;
  box-shadow: var(--atest-shadow-sm);
  color: var(--el-color-white);
  padding: 0;
}

.generate-btn:hover:not(:disabled) {
  transform: translateY(-2px);
  background: var(--el-color-primary-light-3);
  box-shadow: var(--atest-shadow-md);
}

.generate-btn:active:not(:disabled) {
  transform: translateY(0);
}

.generate-btn:disabled {
  background: var(--el-fill-color);
  box-shadow: none;
  color: var(--atest-text-placeholder);
}

.generate-btn.is-loading {
  background: var(--atest-color-accent);
  opacity: 0.8;
}

.chat-input.is-disabled .prompt-input :deep(.el-textarea__inner) {
  background-color: color-mix(in srgb, var(--atest-bg-surface) 85%, #000 15%);
  color: var(--atest-text-placeholder);
}

.chat-input.is-disabled .generate-btn {
  opacity: 0.5;
  cursor: not-allowed;
}

.status-banner {
  display: flex;
  align-items: center;
  gap: var(--atest-spacing-xs);
  padding: 10px 14px;
  border-radius: var(--atest-radius-md);
  border: 1px solid var(--atest-border-color);
  background: var(--atest-bg-elevated);
  color: var(--atest-text-regular);
}

.status-banner.connecting {
  border-color: var(--atest-color-accent);
  color: var(--atest-text-primary);
}

.status-banner.disconnected {
  background: var(--atest-color-danger-soft);
  border-color: rgba(245, 108, 108, 0.32);
  color: var(--el-color-danger, #f56c6c);
}

.status-banner.setup {
  background: var(--atest-color-accent-soft);
  border-color: var(--atest-color-accent-soft);
  color: var(--atest-color-accent);
}

.status-icon {
  display: flex;
  align-items: center;
}

.status-icon.spin {
  animation: spin 1.2s linear infinite;
}

.banner-action {
  margin-left: auto;
}

.banner-text {
  flex: 1;
  line-height: 1.4;
}

@media (max-width: 1024px) {
  .chat-input {
    padding: 18px 28px 22px;
  }

  .action-buttons {
    right: 18px;
  }
}

@media (max-width: 768px) {
  .chat-input {
    padding: 16px 20px;
  }

  .input-box {
    display: flex;
    flex-direction: column;
    gap: 12px;
  }

  .input-box :deep(.el-textarea__inner) {
    padding-right: 16px;
    min-height: 112px;
  }

  .action-buttons {
    position: static;
    flex-direction: row;
    justify-content: flex-end;
    width: 100%;
    gap: 10px;
  }

  .icon-btn {
    width: 44px;
    height: 44px;
  }
}

@media (max-width: 480px) {
  .chat-input {
    padding: 14px 16px;
  }

  .icon-btn {
    width: 40px;
    height: 40px;
  }

  .input-box :deep(.el-textarea__inner) {
    min-height: 100px;
  }
}

.status-fade-enter-active,
.status-fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}

.status-fade-enter-from,
.status-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}
</style>
