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

  // Store models separately for each provider to avoid cross-contamination
  const modelsByProvider = ref<Record<string, Model[]>>({
    ollama: [],
    openai: [],
    deepseek: []
  })

  // Computed property to get models for current provider
  const availableModels = computed(() => {
    return modelsByProvider.value[config.value.provider] || []
  })

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
  watch(() => config.value.provider, async (newProvider, oldProvider) => {
    if (newProvider !== oldProvider) {
      await refreshModels()

      // Validate current model selection for new provider
      const models = modelsByProvider.value[newProvider] || []
      const currentModel = config.value.model
      const modelExists = models.some(m => m.id === currentModel)

      // If current model doesn't exist in new provider, clear selection
      if (!modelExists && currentModel) {
        config.value.model = models.length > 0 ? models[0].id : ''
      }
    }
  })

  /**
   * Refresh available models for current provider
   */
  async function refreshModels() {
    const provider = config.value.provider
    try {
      // Fetch and store models for this specific provider
      const models = await aiService.fetchModels(provider)
      modelsByProvider.value[provider] = models

      // Auto-select first model if none selected
      if (!config.value.model && models.length > 0) {
        config.value.model = models[0].id
      }
    } catch (error) {
      console.error('Failed to fetch models:', error)
      // Use mock models as fallback for this provider
      modelsByProvider.value[provider] = getMockModels(provider)
    }
  }

  /**
   * Handle query submission
   */
  async function handleQuery(prompt: string, options: { includeExplanation: boolean }) {
    console.log('🎯 [useAIChat] handleQuery called', {
      prompt,
      options,
      isConfigured: isConfigured.value,
      config: {
        provider: config.value.provider,
        endpoint: config.value.endpoint,
        model: config.value.model,
        hasApiKey: !!config.value.apiKey
      }
    })

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
      console.log('🚀 [useAIChat] Sending generateSQL request...')

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

      console.log('✅ [useAIChat] Received response', {
        success: response.success,
        hasSql: !!response.sql,
        hasError: !!response.error,
        meta: response.meta
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
        const errorMsg = response.error || 'Failed to generate SQL'
        console.error('❌ [useAIChat] Response failed', {
          success: response.success,
          sql: response.sql,
          error: response.error
        })
        throw new Error(errorMsg)
      }
    } catch (error) {
      console.error('💥 [useAIChat] Exception caught', {
        error,
        message: (error as Error).message,
        stack: (error as Error).stack
      })

      // Add error message
      messages.value.push({
        id: generateId(),
        type: 'error',
        content: `Error: ${(error as Error).message}`,
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
      const result = await aiService.testConnection(config.value)
      config.value.status = result.success ? 'connected' : 'disconnected'
      return result
    } catch (error) {
      config.value.status = 'disconnected'
      return {
        success: false,
        message: (error as Error).message || 'Connection failed',
        provider: config.value.provider,
        error: (error as Error).message
      }
    }
  }

  // Initialize: load models on mount
  refreshModels()

  // Diagnostic logging on startup
  console.log('🚀 [useAIChat] Initialized', {
    provider: config.value.provider,
    endpoint: config.value.endpoint,
    model: config.value.model,
    hasApiKey: !!config.value.apiKey,
    isConfigured: isConfigured.value,
    localStorageGlobal: localStorage.getItem('atest-ai-global-config'),
    localStorageProvider: localStorage.getItem(`atest-ai-config-${config.value.provider}`)
  })

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
