import type { AIConfig, Model } from '@/types'

/**
 * Load configuration from localStorage
 */
export function loadConfig(): AIConfig {
  const globalConfig = localStorage.getItem('atest-ai-global-config')
  let provider: 'ollama' | 'openai' | 'deepseek' = 'ollama'

  if (globalConfig) {
    const parsed = JSON.parse(globalConfig)
    provider = parsed.provider || provider
  }

  const providerConfig = localStorage.getItem(`atest-ai-config-${provider}`)
  const defaults = getDefaultConfig(provider)

  if (providerConfig) {
    return { ...defaults, ...JSON.parse(providerConfig), provider } as AIConfig
  }

  return { ...defaults, provider } as AIConfig
}

/**
 * Save configuration to localStorage
 * Note: Language is managed by main app, not saved here
 */
export function saveConfig(config: AIConfig): void {
  // Save global config (only provider)
  localStorage.setItem('atest-ai-global-config', JSON.stringify({
    provider: config.provider
  }))

  // Save provider-specific config
  const providerConfig = { ...config }
  delete (providerConfig as any).provider

  localStorage.setItem(
    `atest-ai-config-${config.provider}`,
    JSON.stringify(providerConfig)
  )
}

/**
 * Get default configuration for provider
 */
export function getDefaultConfig(provider: string): Partial<AIConfig> {
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
export function getMockModels(provider: string): Model[] {
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
export function generateId(): string {
  return `${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}
