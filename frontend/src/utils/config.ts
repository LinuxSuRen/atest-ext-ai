import type { AIConfig, Model, DatabaseDialect } from '@/types'

export type Provider = 'ollama' | 'openai' | 'deepseek'

/**
 * Load configuration from localStorage
 */
export function loadConfig(): AIConfig {
  const globalConfig = localStorage.getItem('atest-ai-global-config')
  let provider: Provider = 'ollama'

  if (globalConfig) {
    const parsed = JSON.parse(globalConfig)
    provider = (parsed.provider as Provider) || provider
  }

  return loadConfigForProvider(provider)
}

/**
 * Load configuration for a specific provider from localStorage
 */
export function loadConfigForProvider(provider: Provider): AIConfig {
  const defaults = getDefaultConfig(provider)
  const providerConfig = localStorage.getItem(`atest-ai-config-${provider}`)
  const stored = providerConfig ? JSON.parse(providerConfig) : {}

  const isLocalEndpoint = (value: unknown) => {
    if (typeof value !== 'string') {
      return false
    }
    const lower = value.trim().toLowerCase()
    return lower.startsWith('http://localhost') ||
      lower.startsWith('http://127.0.0.1') ||
      lower.startsWith('https://localhost') ||
      lower.startsWith('https://127.0.0.1')
  }

  const config: AIConfig = {
    provider,
    endpoint: (() => {
      const value = stored.endpoint ?? defaults.endpoint ?? ''
      const fallback = defaults.endpoint ?? ''
      if (provider !== 'ollama' && isLocalEndpoint(value)) {
        return normalizeEndpoint(provider, String(fallback))
      }
      if (!value) {
        return normalizeEndpoint(provider, String(fallback))
      }
      return normalizeEndpoint(provider, String(value))
    })(),
    model: stored.model ?? defaults.model ?? '',
    apiKey: stored.apiKey ?? defaults.apiKey ?? '',
    timeout: Number.isFinite(stored.timeout) ? Number(stored.timeout) : (defaults.timeout ?? 120),
    maxTokens: stored.maxTokens ?? defaults.maxTokens ?? 2048,
    status: stored.status ?? 'disconnected',
    databaseDialect: (stored.databaseDialect ?? defaults.databaseDialect ?? 'mysql') as DatabaseDialect
  }

  return config
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
  const { provider, status, ...rest } = config
  const normalizedProvider = (provider === 'local' ? 'ollama' : provider) as Provider
  const defaults = getDefaultConfig(normalizedProvider)
  const providerConfig = {
    endpoint: normalizeEndpoint(normalizedProvider, (rest.endpoint && String(rest.endpoint).trim()) || defaults.endpoint || ''),
    model: rest.model ?? defaults.model ?? '',
    apiKey: rest.apiKey ?? defaults.apiKey ?? '',
    timeout: (typeof rest.timeout === 'number' && rest.timeout > 0 ? rest.timeout : defaults.timeout ?? 120),
    maxTokens: rest.maxTokens ?? defaults.maxTokens ?? 2048,
    databaseDialect: (rest.databaseDialect ?? defaults.databaseDialect ?? 'mysql')
  }

  localStorage.setItem(
    `atest-ai-config-${normalizedProvider}`,
    JSON.stringify(providerConfig)
  )
}

/**
 * Get default configuration for provider
 */
export function getDefaultConfig(provider: string): Partial<AIConfig> {
  const defaults: Record<string, Partial<AIConfig>> = {
    ollama: { endpoint: 'http://localhost:11434', apiKey: '', timeout: 120 },
    openai: { endpoint: 'https://api.openai.com', apiKey: '', timeout: 120 },
    deepseek: { endpoint: 'https://api.deepseek.com', apiKey: '', timeout: 180 }
  }

  return {
    ...(defaults[provider] || defaults.ollama),
    model: '',
    timeout: (defaults[provider] || defaults.ollama)?.timeout ?? 120,
    maxTokens: 2048,
    status: 'disconnected',
    databaseDialect: 'mysql'
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
      { id: 'gpt-5', name: 'GPT-5 ⭐', size: 'Cloud' },
      { id: 'gpt-5-mini', name: 'GPT-5 Mini', size: 'Cloud' },
      { id: 'gpt-5-nano', name: 'GPT-5 Nano', size: 'Cloud' },
      { id: 'gpt-5-pro', name: 'GPT-5 Pro', size: 'Cloud' },
      { id: 'gpt-4.1', name: 'GPT-4.1', size: 'Cloud' }
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

function normalizeEndpoint(provider: Provider, endpoint: string): string {
  const normalizedProvider: Provider = (provider === 'local' ? 'ollama' : provider) as Provider
  if (!endpoint) {
    return endpoint
  }

  let value = endpoint.trim()
  while (value.endsWith('/')) {
    value = value.slice(0, -1)
  }

  if (normalizedProvider === 'openai' || normalizedProvider === 'deepseek') {
    const lower = value.toLowerCase()
    if (lower.endsWith('/v1')) {
      value = value.slice(0, value.length - 3)
      while (value.endsWith('/')) {
        value = value.slice(0, -1)
      }
    }
  }

  return value
}
