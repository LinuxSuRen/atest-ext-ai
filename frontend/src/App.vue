<template>
  <div class="ai-chat-container">
    <AIChatHeader
      :provider="config.provider"
      :status="config.status"
    />

    <div v-if="!isConfigured" class="welcome-panel">
      <AIWelcomePanel @configure="showSettings = true" />
    </div>

    <div v-else class="chat-content">
      <AIChatMessages :messages="messages" />
      <AIChatInput
        :loading="isLoading"
        :include-explanation="includeExplanation"
        @submit="handleQuery"
        @open-settings="showSettings = true"
      />
    </div>

    <AISettingsPanel
      v-model:visible="showSettings"
      :config="config"
      :available-models="availableModels"
      :models-map="modelsByProvider"
      v-model:include-explanation="includeExplanation"
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
import { createTranslator } from './utils/i18n'

// Props passed from main.ts
interface Props {
  context: AppContext
}
const props = defineProps<Props>()

// Wrap host i18n with plugin fallbacks
const pluginContext: AppContext = {
  ...props.context,
  i18n: {
    ...props.context.i18n,
    t: createTranslator(props.context.i18n)
  }
}

// Provide context to all child components
provide('appContext', pluginContext)

// Use composable with context
const {
  config,
  isConfigured,
  availableModels,
  modelsByProvider,
  messages,
  isLoading,
  handleQuery,
  handleSaveConfig,
  handleTestConnection,
  refreshModels
} = useAIChat(pluginContext)

// UI state
const showSettings = ref(false)
const includeExplanation = ref(false)

// Get translation function from context
const { t } = pluginContext.i18n

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
  const result = await handleTestConnection()

  if (result.success) {
    ElMessage.success(t('ai.message.connectionSuccess'))
  } else {
    // Show detailed error message
    let errorMsg = result.message || t('ai.message.connectionFailed')

    // Add helpful tips for Ollama connection issues
    if (result.provider === 'ollama' && result.error) {
      errorMsg += '\n\nTroubleshooting tips:\n' +
        '• Make sure Ollama is running: ollama serve\n' +
        '• Verify endpoint is correct (default: http://localhost:11434)\n' +
        '• Check if firewall is blocking the connection'
    }

    ElMessage({
      message: errorMsg,
      type: 'error',
      duration: 5000,
      dangerouslyUseHTMLString: false,
      customClass: 'connection-error-message'
    })
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
