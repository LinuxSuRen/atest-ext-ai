<template>
  <div :class="['chat-input', { 'is-disabled': props.disabled }]">
    <transition name="status-fade">
      <div v-if="showStatusBanner" class="status-banner" :class="props.status">
        <el-icon v-if="statusIcon" class="status-icon">
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

    <div class="input-shell">
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
    </div>

    <div class="input-footer">
      <div class="footer-left">
        <el-tooltip :content="dialectTooltip" placement="top">
          <el-select
            v-model="dialectModel"
            class="dialect-select"
            size="small"
            :disabled="props.loading"
          >
            <el-option
              v-for="option in props.dialectOptions"
              :key="option.value"
              :label="option.label"
              :value="option.value"
            />
          </el-select>
        </el-tooltip>
        <el-tooltip :content="configureTooltip" placement="top">
          <el-button
            class="footer-btn configure-btn"
            type="default"
            :disabled="props.loading"
            @click="emit('open-settings')"
          >
            <el-icon><Setting /></el-icon>
          </el-button>
        </el-tooltip>
      </div>
      <div class="footer-right">
        <el-tooltip :content="generateTooltip" placement="top">
          <el-button
            class="footer-btn generate-btn"
            type="primary"
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
import { ref, inject, computed, withDefaults } from 'vue'
import { Promotion, Setting, WarningFilled, InfoFilled } from '@element-plus/icons-vue'
import type { AppContext, DatabaseDialect } from '../types'

interface Props {
  loading: boolean
  includeExplanation: boolean
  provider: string
  status: 'connected' | 'disconnected' | 'connecting' | 'setup'
  disabled: boolean
  databaseDialect: DatabaseDialect
  dialectOptions: Array<{ value: DatabaseDialect; label: string }>
}
const props = withDefaults(defineProps<Props>(), {
  dialectOptions: () => [
    { value: 'mysql' as DatabaseDialect, label: 'MySQL' },
    { value: 'postgresql' as DatabaseDialect, label: 'PostgreSQL' },
    { value: 'sqlite' as DatabaseDialect, label: 'SQLite' }
  ]
})

interface Emits {
  (e: 'submit', prompt: string, options: { includeExplanation: boolean; databaseDialect: DatabaseDialect }): void
  (e: 'open-settings'): void
  (e: 'update:databaseDialect', dialect: DatabaseDialect): void
}
const emit = defineEmits<Emits>()

// Inject context
const context = inject<AppContext>('appContext')!
const { t } = context.i18n

// Input state
const prompt = ref('')

const configureTooltip = computed(() => t('ai.tooltip.configure'))
const generateTooltip = computed(() => (props.loading ? t('ai.message.generating') : t('ai.tooltip.generate')))
const dialectTooltip = computed(() => t('ai.tooltip.dialect'))

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
    case 'disconnected':
      return WarningFilled
    case 'setup':
      return InfoFilled
    default:
      return null
  }
})

const showStatusBanner = computed(() => props.status === 'disconnected' || props.status === 'setup')
const showConfigureLink = showStatusBanner

const dialectModel = computed({
  get: () => props.databaseDialect,
  set: (value: DatabaseDialect) => emit('update:databaseDialect', value)
})

function handleSubmit() {
  if (!prompt.value.trim() || props.loading || props.disabled) return

  emit('submit', prompt.value, {
    includeExplanation: props.includeExplanation,
    databaseDialect: props.databaseDialect
  })

  // Clear input after submit
  prompt.value = ''
}
</script>

<style scoped>
.chat-input {
  flex-shrink: 0;
  padding: clamp(16px, 3vw, 20px) clamp(20px, 4vw, 40px) clamp(20px, 3vw, 24px);
  background: var(--atest-bg-surface);
  border-top: 1px solid var(--atest-border-color);
  display: flex;
  flex-direction: column;
  gap: var(--atest-spacing-sm);
}

.input-shell {
  position: relative;
}

.prompt-input {
  display: block;
}

.input-shell :deep(.el-textarea__inner) {
  border-radius: 12px;
  border: 2px solid var(--atest-border-color);
  padding: 12px 16px;
  font-size: 14px;
  line-height: 1.6;
  resize: none;
  transition: var(--atest-transition-base);
  box-shadow: 0 2px 8px var(--el-box-shadow-lighter);
  min-height: 124px;
}

.input-shell :deep(.el-textarea__inner:focus) {
  border-color: var(--atest-color-accent);
  box-shadow: 0 0 0 3px var(--atest-color-accent-soft);
}

.input-shell :deep(.el-textarea__inner::placeholder) {
  color: var(--atest-text-placeholder);
}
.chat-input.is-disabled .prompt-input :deep(.el-textarea__inner) {
  background-color: color-mix(in srgb, var(--atest-bg-surface) 85%, #000 15%);
  color: var(--atest-text-placeholder);
}

.chat-input.is-disabled .generate-btn {
  opacity: 0.5;
  cursor: not-allowed;
}

.input-footer {
  margin-top: var(--atest-spacing-sm);
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: var(--atest-spacing-sm);
}

.footer-left,
.footer-right {
  display: flex;
  align-items: center;
  gap: var(--atest-spacing-xs);
}

.footer-left {
  justify-content: flex-start;
}

.footer-right {
  justify-content: flex-end;
}

.dialect-select {
  min-width: 170px;
}

.dialect-select :deep(.el-select__wrapper),
.dialect-select :deep(.el-input__wrapper) {
  border-radius: 10px;
}

.footer-btn {
  display: inline-flex;
  justify-content: center;
  align-items: center;
  width: 48px;
  height: 48px;
  border-radius: 12px;
  transition: var(--atest-transition-base);
}

.configure-btn {
  border: 1px solid var(--atest-border-color);
  background: var(--atest-bg-elevated);
  color: var(--atest-color-accent);
}

.configure-btn:hover:not(:disabled) {
  border-color: var(--atest-color-accent);
  background: var(--atest-color-accent-soft);
}

.generate-btn {
  background: var(--atest-color-accent);
  color: #fff;
}

.generate-btn:hover:not(:disabled) {
  background: var(--el-color-primary-light-3);
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

  .input-shell :deep(.el-textarea__inner) {
    min-height: 112px;
  }

  .input-footer {
    grid-template-columns: 1fr;
  }

  .footer-left,
  .footer-right {
    justify-content: space-between;
  }

  .dialect-select {
    flex: 1;
    min-width: 0;
  }

  .footer-btn {
    width: 44px;
    height: 44px;
  }
}

@media (max-width: 480px) {
  .chat-input {
    padding: 14px 16px;
  }

  .footer-btn {
    width: 40px;
    height: 40px;
  }

  .input-shell :deep(.el-textarea__inner) {
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
</style>
