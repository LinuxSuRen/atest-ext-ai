import { ref, computed, watch } from 'vue'
import type { AppContext, AIConfig, Message, Model, DatabaseDialect } from '@/types'
import { loadConfig, loadConfigForProvider, saveConfig, generateId, type Provider } from '@/utils/config'
import { aiService } from '@/services/aiService'

/**
 * Main composable for AI Chat functionality
 * Uses aiService for API calls and manages UI state
 */
export function useAIChat(_context: AppContext) {
  // Note: context parameter is kept for future use (e.g., authentication tokens)

  // Configuration management
  const config = ref<AIConfig>(loadConfig())

  const resolveProviderKey = (provider: string): Provider => {
    if (provider === 'local') {
      return 'ollama'
    }
    return provider as Provider
  }

  // Store models separately for each provider to avoid cross-contamination
  const modelsByProvider = ref<Record<string, Model[]>>({
    ollama: [],
    openai: [],
    deepseek: []
  })

  const catalogCache = ref<Record<string, Model[]>>({})

  async function initializeModelCatalog() {
    try {
      const catalog = await aiService.fetchModelCatalog()
      const normalizedCatalog: Record<string, Model[]> = {}

      Object.entries(catalog).forEach(([provider, entry]) => {
        normalizedCatalog[provider] = entry.models || []
      })

      catalogCache.value = normalizedCatalog

      for (const [provider, models] of Object.entries(normalizedCatalog)) {
        if (!modelsByProvider.value[provider]) {
          modelsByProvider.value[provider] = models
        } else if ((modelsByProvider.value[provider] || []).length === 0 && models.length > 0) {
          modelsByProvider.value[provider] = models
        }
      }
    } catch (error) {
      console.error('Failed to load model catalog', error)
    }
  }

  void initializeModelCatalog()

  // Computed property to get models for current provider
  const availableModels = computed(() => {
    const key = resolveProviderKey(config.value.provider)
    return modelsByProvider.value[key] || []
  })

  // Check if AI is properly configured
  const isConfigured = computed(() => {
    const c = config.value
    const providerKey = resolveProviderKey(c.provider)
    if (providerKey === 'ollama') {
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
    if (newProvider === oldProvider) {
      return
    }

    const normalizedProvider = resolveProviderKey(newProvider)
    const providerConfig = loadConfigForProvider(normalizedProvider)

    config.value = {
      ...config.value,
      ...providerConfig,
      provider: newProvider,
      status: providerConfig.status ?? 'disconnected'
    }

    await refreshModels(normalizedProvider)

    // Validate current model selection for new provider
    const models = modelsByProvider.value[normalizedProvider] || []
    const currentModel = config.value.model
    const modelExists = models.some(m => m.id === currentModel)

    if (!modelExists) {
      config.value.model = models.length > 0 ? models[0].id : ''
    }
  })

  /**
   * Refresh available models for current provider
   */
  async function refreshModels(targetProvider?: string) {
    const provider = targetProvider ?? config.value.provider
    const storeKey = resolveProviderKey(provider)

    try {
      // Fetch and store models for this specific provider
      const models = await aiService.fetchModels(provider)
      modelsByProvider.value[storeKey] = models

      // Auto-select first model if none selected and refreshing active provider
      if (storeKey === resolveProviderKey(config.value.provider) && !config.value.model && models.length > 0) {
        config.value.model = models[0].id
      }
    } catch (error) {
      console.error('Failed to fetch models:', error)
      const cachedFallback = catalogCache.value[storeKey]
      if (cachedFallback && cachedFallback.length) {
        modelsByProvider.value[storeKey] = cachedFallback
        return
      }

      try {
        const catalog = await aiService.fetchModelCatalog(storeKey)
        const entry = catalog[storeKey]
        const fallbackModels = entry?.models ?? []
        modelsByProvider.value[storeKey] = fallbackModels
        if (fallbackModels.length) {
          catalogCache.value[storeKey] = fallbackModels
        }
      } catch (catalogError) {
        console.error('Failed to fetch catalog fallback:', catalogError)
        modelsByProvider.value[storeKey] = []
      }
    }
  }

  /**
   * Handle query submission
   */
  async function handleQuery(prompt: string, options: { includeExplanation: boolean; databaseDialect: DatabaseDialect }) {
    console.log('🎯 [useAIChat] handleQuery called', {
      prompt,
      options,
      isConfigured: isConfigured.value,
      config: {
        provider: config.value.provider,
        endpoint: config.value.endpoint,
        model: config.value.model,
        hasApiKey: !!config.value.apiKey,
        timeout: config.value.timeout
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
    config.value.status = 'connecting'
    try {
      console.log('🚀 [useAIChat] Sending generateSQL request...')

      const response = await aiService.generateSQL({
        provider: config.value.provider,
        endpoint: config.value.endpoint,
        apiKey: config.value.apiKey,
        model: config.value.model,
        prompt,
        timeout: config.value.timeout,
        maxTokens: config.value.maxTokens,
        includeExplanation: options.includeExplanation,
        databaseDialect: options.databaseDialect ?? config.value.databaseDialect ?? 'mysql'
      })

      console.log('✅ [useAIChat] Received response', {
        success: response.success,
        hasSql: !!response.sql,
        hasError: !!response.error,
        meta: response.meta
      })

      if (response.success && response.sql) {
        config.value.status = 'connected'
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
      config.value.status = 'disconnected'
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
  async function handleTestConnection(testConfig?: AIConfig) {
    config.value.status = 'connecting'
    const payload: AIConfig = {
      ...config.value,
      ...(testConfig ?? {})
    }
    try {
      const result = await aiService.testConnection(payload)
      config.value.status = result.success ? 'connected' : 'disconnected'
      return result
    } catch (error) {
      config.value.status = 'disconnected'
      return {
        success: false,
        message: (error as Error).message || 'Connection failed',
        provider: payload.provider,
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
    modelsByProvider,
    messages,
    isLoading,
    handleQuery,
    handleSaveConfig,
    handleTestConnection,
    refreshModels
  }
}
