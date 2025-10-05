import { ref, computed, watch } from 'vue'
import type { AppContext, AIConfig, Message, Model } from '@/types'
import { loadConfig, saveConfig, getMockModels, generateId } from '@/utils/config'

const API_STORE = 'ai'

/**
 * Main composable for AI Chat functionality
 * Uses context from main app for API calls and i18n
 */
export function useAIChat(_context: AppContext) {
  // Note: We use fetch API directly instead of context.API.DataQuery
  // because DataQuery is designed for database queries and transforms the request format
  // The context parameter is kept for future use (e.g., authentication tokens)

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
      const result = await callAPI<{ models: Model[] }>('models', {
        provider: config.value.provider
      })
      availableModels.value = result.models || []

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
      const result = await callAPI<{
        content: string
        meta: string
        success: string
        error?: string
      }>('generate', {
        model: config.value.model,
        prompt,
        config: JSON.stringify({
          include_explanation: options.includeExplanation,
          provider: config.value.provider,
          endpoint: config.value.endpoint,
          api_key: config.value.apiKey,
          temperature: config.value.temperature,
          max_tokens: config.value.maxTokens
        })
      })

      if (result.success === 'true' && result.content) {
        // Add AI response
        messages.value.push({
          id: generateId(),
          type: 'ai',
          content: 'Generated SQL:',
          sql: result.content,
          meta: result.meta ? JSON.parse(result.meta) : undefined,
          timestamp: Date.now()
        })
      } else {
        throw new Error(result.error || 'Failed to generate SQL')
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
      await callAPI('update_config', {
        provider: config.value.provider,
        config: {
          provider: config.value.provider,
          endpoint: config.value.endpoint,
          model: config.value.model,
          api_key: config.value.apiKey,
          temperature: config.value.temperature,
          max_tokens: config.value.maxTokens
        }
      })
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
      const result = await callAPI<{ success: string }>('test_connection', config.value)
      config.value.status = result.success === 'true' ? 'connected' : 'disconnected'
      return { success: result.success === 'true' }
    } catch (error) {
      config.value.status = 'disconnected'
      throw error
    }
  }

  /**
   * Call backend API directly
   *
   * Note: We use fetch directly instead of DataQuery because DataQuery
   * is designed for database queries and transforms the request format.
   * The AI plugin expects: {type: 'ai', key: 'operation', sql: 'params_json'}
   */
  async function callAPI<T>(key: string, data: any): Promise<T> {
    const response = await fetch('/api/v1/data/query', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Store-Name': API_STORE
      },
      body: JSON.stringify({
        type: 'ai',
        key,
        sql: JSON.stringify(data)
      })
    })

    if (!response.ok) {
      throw new Error(`API error: ${response.status}`)
    }

    const result = await response.json()

    // Parse key-value pair format from backend
    const parsed: any = {}
    if (result.data) {
      for (const pair of result.data) {
        try {
          parsed[pair.key] = JSON.parse(pair.value)
        } catch {
          parsed[pair.key] = pair.value
        }
      }
    }

    return parsed as T
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
