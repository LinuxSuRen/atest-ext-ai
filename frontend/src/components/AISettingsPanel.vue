<template>
  <el-drawer
    :model-value="props.visible"
    :title="t('ai.settings.title')"
    size="500px"
    @close="emit('update:visible', false)"
  >
    <el-form :model="localConfig" label-width="120px">
      <!-- Provider Selection -->
      <el-form-item :label="t('ai.settings.provider')">
        <el-select v-model="localConfig.provider">
          <el-option
            value="ollama"
            :label="t('ai.provider.ollama.name')"
          >
            <div>
              <div>{{ t('ai.provider.ollama.name') }}</div>
              <div style="font-size: 12px; color: #909399">
                {{ t('ai.provider.ollama.description') }}
              </div>
            </div>
          </el-option>
          <el-option
            value="openai"
            :label="t('ai.provider.openai.name')"
          >
            <div>
              <div>{{ t('ai.provider.openai.name') }}</div>
              <div style="font-size: 12px; color: #909399">
                {{ t('ai.provider.openai.description') }}
              </div>
            </div>
          </el-option>
          <el-option
            value="deepseek"
            :label="t('ai.provider.deepseek.name')"
          >
            <div>
              <div>{{ t('ai.provider.deepseek.name') }}</div>
              <div style="font-size: 12px; color: #909399">
                {{ t('ai.provider.deepseek.description') }}
              </div>
            </div>
          </el-option>
        </el-select>
      </el-form-item>

      <!-- Endpoint -->
      <el-form-item :label="t('ai.settings.endpoint')">
        <el-input v-model="localConfig.endpoint" />
      </el-form-item>

      <!-- Model Selection -->
      <el-form-item :label="t('ai.settings.model')">
        <el-select
          v-model="localConfig.model"
          :placeholder="t('ai.welcome.noModels')"
        >
          <template #prefix>
            <el-button
              link
              type="primary"
              @click="emit('refresh-models')"
            >
              <el-icon><Refresh /></el-icon>
            </el-button>
          </template>
          <el-option
            v-for="model in props.availableModels"
            :key="model.id"
            :value="model.id"
            :label="model.name"
          >
            <div style="display: flex; justify-content: space-between">
              <span>{{ model.name }}</span>
              <span style="font-size: 12px; color: #909399">{{ model.size }}</span>
            </div>
          </el-option>
        </el-select>
      </el-form-item>

      <!-- API Key (for cloud services) -->
      <el-form-item
        v-if="localConfig.provider !== 'ollama'"
        :label="t('ai.settings.apiKey')"
      >
        <el-input
          v-model="localConfig.apiKey"
          type="password"
          show-password
        />
      </el-form-item>

      <!-- Advanced Settings -->
      <el-divider>{{ t('ai.settings.advanced') }}</el-divider>

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

    <template #footer>
      <div class="dialog-footer">
        <el-button @click="emit('update:visible', false)">
          {{ t('ai.button.close') }}
        </el-button>
        <el-button @click="emit('test-connection')">
          {{ t('ai.button.testConnection') }}
        </el-button>
        <el-button type="primary" @click="handleSave">
          {{ t('ai.button.save') }}
        </el-button>
      </div>
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, inject, watch } from 'vue'
import { Refresh } from '@element-plus/icons-vue'
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
.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

:deep(.el-select-dropdown__item) {
  height: auto;
  line-height: 1.5;
  padding: 8px 16px;
}
</style>
