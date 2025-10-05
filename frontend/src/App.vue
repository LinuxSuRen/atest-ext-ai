<template>
  <div class="ai-chat-container">
    <AIChatHeader
      :provider="config.provider"
      :status="config.status"
      @open-settings="showSettings = true"
    />

    <div v-if="!isConfigured" class="welcome-panel">
      <AIWelcomePanel @configure="showSettings = true" />
    </div>

    <div v-else class="chat-content">
      <AIChatMessages :messages="messages" />
      <AIChatInput
        :loading="isLoading"
        @submit="handleQuery"
      />
    </div>

    <AISettingsPanel
      v-model:visible="showSettings"
      :config="config"
      :available-models="availableModels"
      @save="handleSave"
      @test-connection="handleTest"
      @refresh-models="refreshModels"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, provide } from 'vue'
import { ElMessage } from 'element-plus'
import type { AppContext } from './types'
import { useAIChat } from './composables/useAIChat'
import AIChatHeader from './components/AIChatHeader.vue'
import AIChatMessages from './components/AIChatMessages.vue'
import AIChatInput from './components/AIChatInput.vue'
import AISettingsPanel from './components/AISettingsPanel.vue'
import AIWelcomePanel from './components/AIWelcomePanel.vue'

// Props passed from main.ts
interface Props {
  context: AppContext
}
const props = defineProps<Props>()

// Provide context to all child components
provide('appContext', props.context)

// Use composable with context
const {
  config,
  isConfigured,
  availableModels,
  messages,
  isLoading,
  handleQuery,
  handleSaveConfig,
  handleTestConnection,
  refreshModels
} = useAIChat(props.context)

// UI state
const showSettings = ref(false)

// Get translation function from context
const { t } = props.context.i18n

// Save configuration
async function handleSave() {
  try {
    await handleSaveConfig()
    ElMessage.success(t('ai.message.configSaved'))
    showSettings.value = false
  } catch (error) {
    ElMessage.warning(t('ai.message.configSaveFailed'))
  }
}

// Test connection
async function handleTest() {
  try {
    const result = await handleTestConnection()
    if (result.success) {
      ElMessage.success(t('ai.message.connectionSuccess'))
    } else {
      ElMessage.error(t('ai.message.connectionFailed'))
    }
  } catch (error) {
    ElMessage.error(t('ai.message.connectionFailed'))
  }
}
</script>

<style scoped>
.ai-chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #f5f7fa;
}

.welcome-panel {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
}

.chat-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
