<template>
  <el-drawer
    :model-value="props.visible"
    :title="t('ai.settings.title')"
    size="550px"
    @close="emit('update:visible', false)"
  >
    <el-tabs v-model="activeTab" class="provider-tabs">
      <!-- Local Services Tab -->
      <el-tab-pane name="local">
        <template #label>
          <div class="tab-label">
            <el-icon><Monitor /></el-icon>
            <span>{{ t('ai.settings.localServices') }}</span>
          </div>
        </template>

        <div class="provider-section">
          <!-- Ollama Provider Card -->
          <el-card class="provider-card" shadow="hover">
            <template #header>
              <div class="provider-header">
                <div class="provider-title">
                  <el-icon :size="20" class="provider-icon"><Cpu /></el-icon>
                  <span>{{ t('ai.provider.ollama.name') }}</span>
                </div>
                <el-tag size="small" type="success" effect="light">
                  {{ t('ai.provider.local') }}
                </el-tag>
              </div>
              <div class="provider-description">
                {{ t('ai.provider.ollama.description') }}
              </div>
            </template>

            <el-form :model="localConfig" label-width="100px" label-position="top">
              <!-- Endpoint -->
              <el-form-item :label="t('ai.settings.endpoint')">
                <el-input
                  v-model="localConfig.endpoint"
                  placeholder="http://localhost:11434"
                />
              </el-form-item>

              <!-- Model Selection -->
              <el-form-item :label="t('ai.settings.model')">
                <div class="model-select-wrapper">
                  <el-select
                    v-model="localConfig.model"
                    :placeholder="t('ai.welcome.noModels')"
                    style="width: 100%"
                  >
                    <el-option
                      v-for="model in props.availableModels"
                      :key="model.id"
                      :value="model.id"
                      :label="model.name"
                    >
                      <div class="model-option">
                        <span class="model-name">{{ model.name }}</span>
                        <span class="model-size">{{ model.size }}</span>
                      </div>
                    </el-option>
                  </el-select>
                  <el-button
                    link
                    type="primary"
                    @click="emit('refresh-models')"
                    class="refresh-btn"
                  >
                    <el-icon><Refresh /></el-icon>
                    {{ t('ai.button.refresh') }}
                  </el-button>
                </div>
              </el-form-item>
            </el-form>
          </el-card>
        </div>
      </el-tab-pane>

      <!-- Cloud Services Tab -->
      <el-tab-pane name="cloud">
        <template #label>
          <div class="tab-label">
            <el-icon><CloudIcon /></el-icon>
            <span>{{ t('ai.settings.cloudServices') }}</span>
          </div>
        </template>

        <div class="provider-section">
          <!-- Provider Radio Group -->
          <el-radio-group v-model="localConfig.provider" class="provider-radio-group">
            <!-- OpenAI -->
            <el-card class="provider-card" :class="{ 'is-selected': localConfig.provider === 'openai' }" shadow="hover">
              <template #header>
                <el-radio value="openai" size="large">
                  <div class="provider-header">
                    <div class="provider-title">
                      <el-icon :size="20" class="provider-icon"><MagicStick /></el-icon>
                      <span>{{ t('ai.provider.openai.name') }}</span>
                    </div>
                    <el-tag size="small" type="primary" effect="light">
                      {{ t('ai.provider.cloud') }}
                    </el-tag>
                  </div>
                </el-radio>
                <div class="provider-description">
                  {{ t('ai.provider.openai.description') }}
                </div>
              </template>

              <el-collapse-transition>
                <el-form v-show="localConfig.provider === 'openai'" :model="localConfig" label-width="100px" label-position="top">
                  <el-form-item :label="t('ai.settings.apiKey')">
                    <el-input
                      v-model="localConfig.apiKey"
                      type="password"
                      show-password
                      placeholder="sk-..."
                    />
                  </el-form-item>
                  <el-form-item :label="t('ai.settings.model')">
                    <el-select v-model="localConfig.model" style="width: 100%">
                      <el-option value="gpt-4" label="GPT-4" />
                      <el-option value="gpt-3.5-turbo" label="GPT-3.5 Turbo" />
                    </el-select>
                  </el-form-item>
                </el-form>
              </el-collapse-transition>
            </el-card>

            <!-- DeepSeek -->
            <el-card class="provider-card" :class="{ 'is-selected': localConfig.provider === 'deepseek' }" shadow="hover">
              <template #header>
                <el-radio value="deepseek" size="large">
                  <div class="provider-header">
                    <div class="provider-title">
                      <el-icon :size="20" class="provider-icon"><Connection /></el-icon>
                      <span>{{ t('ai.provider.deepseek.name') }}</span>
                    </div>
                    <el-tag size="small" type="warning" effect="light">
                      {{ t('ai.provider.cloud') }}
                    </el-tag>
                  </div>
                </el-radio>
                <div class="provider-description">
                  {{ t('ai.provider.deepseek.description') }}
                </div>
              </template>

              <el-collapse-transition>
                <el-form v-show="localConfig.provider === 'deepseek'" :model="localConfig" label-width="100px" label-position="top">
                  <el-form-item :label="t('ai.settings.apiKey')">
                    <el-input
                      v-model="localConfig.apiKey"
                      type="password"
                      show-password
                      placeholder="sk-..."
                    />
                  </el-form-item>
                  <el-form-item :label="t('ai.settings.model')">
                    <el-select v-model="localConfig.model" style="width: 100%">
                      <el-option value="deepseek-coder" label="DeepSeek Coder" />
                      <el-option value="deepseek-chat" label="DeepSeek Chat" />
                    </el-select>
                  </el-form-item>
                </el-form>
              </el-collapse-transition>
            </el-card>
          </el-radio-group>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- Advanced Settings (Common for all providers) -->
    <div class="advanced-section">
      <el-divider>{{ t('ai.settings.advanced') }}</el-divider>
      <el-form :model="localConfig" label-width="120px">
        <el-form-item :label="t('ai.settings.temperature')">
          <el-slider
            v-model="localConfig.temperature"
            :min="0"
            :max="1"
            :step="0.1"
            show-input
          />
        </el-form-item>

        <el-form-item :label="t('ai.settings.maxTokens')">
          <el-input-number
            v-model="localConfig.maxTokens"
            :min="256"
            :max="8192"
            :step="256"
          />
        </el-form-item>
      </el-form>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="emit('update:visible', false)">
          {{ t('ai.button.close') }}
        </el-button>
        <el-button @click="emit('test-connection')">
          <el-icon><Connection /></el-icon>
          {{ t('ai.button.testConnection') }}
        </el-button>
        <el-button type="primary" @click="handleSave">
          <el-icon><Check /></el-icon>
          {{ t('ai.button.save') }}
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, inject, watch, computed } from 'vue'
import {
  Refresh,
  Monitor,
  Cloudy as CloudIcon,
  Cpu,
  MagicStick,
  Connection,
  Check
} from '@element-plus/icons-vue'
import type { AppContext, AIConfig, Model } from '../types'

interface Props {
  visible: boolean
  config: AIConfig
  availableModels: Model[]
}
const props = defineProps<Props>()

interface Emits {
  (e: 'update:visible', value: boolean): void
  (e: 'save'): void
  (e: 'test-connection'): void
  (e: 'refresh-models'): void
}
const emit = defineEmits<Emits>()

// Inject context
const context = inject<AppContext>('appContext')!
const { t } = context.i18n

// Local config copy
const localConfig = ref<AIConfig>({ ...props.config })

// Active tab state - automatically switch based on provider
const activeTab = computed({
  get() {
    return localConfig.value.provider === 'ollama' ? 'local' : 'cloud'
  },
  set(tab: string) {
    // When switching tabs, update provider to a default for that category
    if (tab === 'local') {
      localConfig.value.provider = 'ollama'
    } else if (tab === 'cloud' && localConfig.value.provider === 'ollama') {
      localConfig.value.provider = 'openai'
    }
  }
})

// Watch config changes
watch(() => props.config, (newConfig) => {
  localConfig.value = { ...newConfig }
}, { deep: true })

function handleSave() {
  // Copy local config back to props
  Object.assign(props.config, localConfig.value)
  emit('save')
}
</script>

<style scoped>
/* Tabs styling */
.provider-tabs {
  margin: -20px -20px 0;
}

.provider-tabs :deep(.el-tabs__header) {
  background: #f5f7fa;
  margin: 0;
  padding: 0 20px;
}

.provider-tabs :deep(.el-tabs__nav-wrap) {
  padding-top: 8px;
}

.provider-tabs :deep(.el-tabs__content) {
  padding: 20px;
}

.tab-label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 500;
}

/* Provider Section */
.provider-section {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

/* Provider Card */
.provider-card {
  border-radius: 12px;
  transition: all 0.3s ease;
}

.provider-card.is-selected {
  border-color: #667eea;
  box-shadow: 0 4px 12px rgba(102, 126, 234, 0.15);
}

.provider-card :deep(.el-card__header) {
  padding: 16px 20px;
  background: linear-gradient(to right, #fafbfc, #f5f7fa);
  border-bottom: 1px solid #e4e7ed;
}

.provider-card :deep(.el-card__body) {
  padding: 20px;
}

/* Provider Header */
.provider-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.provider-title {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.provider-icon {
  color: #667eea;
}

.provider-description {
  font-size: 13px;
  color: #606266;
  margin-top: 8px;
  line-height: 1.5;
}

/* Provider Radio Group */
.provider-radio-group {
  display: flex;
  flex-direction: column;
  gap: 16px;
  width: 100%;
}

.provider-radio-group :deep(.el-radio) {
  margin-right: 0;
}

.provider-radio-group :deep(.el-radio__label) {
  width: 100%;
  padding-left: 0;
}

/* Model Select */
.model-select-wrapper {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.refresh-btn {
  align-self: flex-start;
  font-size: 13px;
}

.model-option {
  display: flex;
  justify-content: space-between;
  align-items: center;
  width: 100%;
}

.model-name {
  font-weight: 500;
}

.model-size {
  font-size: 12px;
  color: #909399;
}

/* Advanced Section */
.advanced-section {
  margin-top: 24px;
  padding: 0 20px;
}

.advanced-section .el-divider {
  margin: 24px 0 20px;
}

/* Dialog Footer */
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
}

.dialog-footer .el-button {
  padding: 10px 20px;
  border-radius: 8px;
}

/* Form styling */
:deep(.el-form-item__label) {
  font-weight: 500;
  color: #606266;
}

:deep(.el-input__inner),
:deep(.el-select) {
  border-radius: 8px;
}

:deep(.el-select-dropdown__item) {
  height: auto;
  line-height: 1.5;
  padding: 10px 16px;
}
</style>
