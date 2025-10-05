import { ref, computed, watch } from 'vue'
import type { AppContext, AIConfig, Message, Model } from '@/types'
import { loadConfig, saveConfig, getMockModels, generateId } from '@/utils/config'
import { aiService } from '@/services/aiService'

/**
 * Main composable for AI Chat functionality
 * Uses aiService for API calls and manages UI state
 */
export function useAIChat(_context: AppContext) {
  // Note: context parameter is kept for future use (e.g., authentication tokens)

  // Configuration management
  const config = ref<AIConfig>(loadConfig())
  const availableModels = ref<Model[]>([])

  // Check if AI is properly configured
  const isConfigured = computed(() => {
    const c = config.value
    if (c.provider === 'ollama') {
      return !!c.endpoint && !!c.model
    }
    return !!c.apiKey && !!c.model
  })

  // Message management
  const messages = ref<Message[]>([])
  const isLoading = ref(false)

  // Watch config changes and auto-save to localStorage
  watch(config, (newConfig) => {
    saveConfig(newConfig)
  }, { deep: true })

  // Watch provider changes and refresh models
  watch(() => config.value.provider, async () => {
    await refreshModels()
  })

  /**
   * Refresh available models for current provider
   */
  async function refreshModels() {
    try {
      availableModels.value = await aiService.fetchModels(config.value.provider)

      // Auto-select first model if none selected
      if (!config.value.model && availableModels.value.length > 0) {
        config.value.model = availableModels.value[0].id
      }
    } catch (error) {
      console.error('Failed to fetch models:', error)
      availableModels.value = getMockModels(config.value.provider)
    }
  }

  /**
   * Handle query submission
   */
  async function handleQuery(prompt: string, options: { includeExplanation: boolean }) {
    // Add user message
    const userMsg: Message = {
      id: generateId(),
      type: 'user',
      content: prompt,
      timestamp: Date.now()
    }
    messages.value.push(userMsg)

    // Show loading
    isLoading.value = true
    try {
      const response = await aiService.generateSQL({
        provider: config.value.provider,
        endpoint: config.value.endpoint,
        apiKey: config.value.apiKey,
        model: config.value.model,
        prompt,
        temperature: config.value.temperature,
        maxTokens: config.value.maxTokens,
        includeExplanation: options.includeExplanation
      })

      if (response.success && response.sql) {
        // Add AI response
        messages.value.push({
          id: generateId(),
          type: 'ai',
          content: 'Generated SQL:',
          sql: response.sql,
          meta: response.meta,
          timestamp: Date.now()
        })
      } else {
        throw new Error(response.error || 'Failed to generate SQL')
      }
    } catch (error) {
      // Add error message
      messages.value.push({
        id: generateId(),
        type: 'error',
        content: (error as Error).message,
        timestamp: Date.now()
      })
    } finally {
      isLoading.value = false
    }
  }

  /**
   * Save configuration to backend
   */
  async function handleSaveConfig() {
    try {
      await aiService.saveConfig(config.value)
      return { success: true }
    } catch (error) {
      console.error('Failed to save config to backend:', error)
      throw error
    }
  }

  /**
   * Test connection to AI provider
   */
  async function handleTestConnection() {
    config.value.status = 'connecting'
    try {
      const success = await aiService.testConnection(config.value)
      config.value.status = success ? 'connected' : 'disconnected'
      return { success }
    } catch (error) {
      config.value.status = 'disconnected'
      throw error
    }
  }

  // Initialize: load models on mount
  refreshModels()

  return {
    config,
    isConfigured,
    availableModels,
    messages,
    isLoading,
    handleQuery,
    handleSaveConfig,
    handleTestConnection,
    refreshModels
  }
}
