import { ref, computed, watch } from 'vue'
import type { AppContext, AIConfig, Message, Model } from '@/types'

const API_STORE = 'ai'

/**
 * Main composable for AI Chat functionality
 * Uses context from main app for API calls and i18n
 */
export function useAIChat(context: AppContext) {
  const { API } = context

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
   * Call backend API using main app's DataQuery
   */
  function callAPI<T>(key: string, data: any): Promise<T> {
    return new Promise((resolve, reject) => {
      API.DataQuery(
        API_STORE,
        'atest-store-orm',
        '',
        JSON.stringify({
          type: 'ai',
          key,
          sql: JSON.stringify(data)
        }),
        (result: any) => {
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
          resolve(parsed as T)
        },
        (error: any) => {
          reject(new Error(error?.message || 'API call failed'))
        }
      )
    })
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

/**
 * Load configuration from localStorage
 */
function loadConfig(): AIConfig {
  const globalConfig = localStorage.getItem('atest-ai-global-config')
  let provider: 'ollama' | 'openai' | 'deepseek' = 'ollama'
  let language: 'en' | 'zh' = 'en'

  if (globalConfig) {
    const parsed = JSON.parse(globalConfig)
    provider = parsed.provider || provider
    language = parsed.language || language
  }

  const providerConfig = localStorage.getItem(`atest-ai-config-${provider}`)
  const defaults = getDefaultConfig(provider)

  if (providerConfig) {
    return { ...defaults, ...JSON.parse(providerConfig), provider, language } as AIConfig
  }

  return { ...defaults, provider, language } as AIConfig
}

/**
 * Save configuration to localStorage
 */
function saveConfig(config: AIConfig) {
  // Save global config (provider and language)
  localStorage.setItem('atest-ai-global-config', JSON.stringify({
    provider: config.provider,
    language: config.language
  }))

  // Save provider-specific config
  const providerConfig = { ...config }
  delete (providerConfig as any).provider
  delete (providerConfig as any).language

  localStorage.setItem(
    `atest-ai-config-${config.provider}`,
    JSON.stringify(providerConfig)
  )
}

/**
 * Get default configuration for provider
 */
function getDefaultConfig(provider: string): Partial<AIConfig> {
  const defaults: Record<string, Partial<AIConfig>> = {
    ollama: { endpoint: 'http://localhost:11434', apiKey: '' },
    openai: { endpoint: '', apiKey: '' },
    deepseek: { endpoint: '', apiKey: '' }
  }

  return {
    ...(defaults[provider] || defaults.ollama),
    model: '',
    temperature: 0.7,
    maxTokens: 2048,
    status: 'disconnected'
  }
}

/**
 * Get mock models when API fails
 */
function getMockModels(provider: string): Model[] {
  const mocks: Record<string, Model[]> = {
    ollama: [
      { id: 'llama3.2:3b', name: 'Llama 3.2 3B', size: '2GB' },
      { id: 'gemma2:9b', name: 'Gemma 2 9B', size: '5GB' }
    ],
    openai: [
      { id: 'gpt-4o', name: 'GPT-4o', size: 'Cloud' },
      { id: 'gpt-4o-mini', name: 'GPT-4o Mini', size: 'Cloud' }
    ],
    deepseek: [
      { id: 'deepseek-chat', name: 'DeepSeek Chat', size: 'Cloud' },
      { id: 'deepseek-reasoner', name: 'DeepSeek Reasoner', size: 'Cloud' }
    ]
  }
  return mocks[provider] || []
}

/**
 * Generate unique ID
 */
function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}
