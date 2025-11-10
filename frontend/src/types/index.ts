import type { Ref } from 'vue'

/**
 * Context passed from main app to plugin
 * Provides access to main app's i18n, API, and Cache
 */
export interface AppContext {
  i18n: {
    t: (key: string) => string
    locale: Ref<string>
  }
  API: APIClient  // Main app's API object from net.ts
  Cache: CacheManager  // Main app's Cache object from cache.ts
}

/**
 * AI configuration
 * Note: Language is managed by main app, not stored in plugin config
 *
 * Provider values:
 * - 'ollama': Local Ollama service (user-facing option)
 * - 'openai': OpenAI cloud service
 * - 'deepseek': DeepSeek cloud service
 * - 'local': Internal alias for 'ollama' (backward compatibility only, not shown in UI)
 */
export type DatabaseDialect = 'mysql' | 'postgresql' | 'sqlite'

export type HTTPMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'

export interface APIRequestOptions {
  path: string
  method?: HTTPMethod
  headers?: Record<string, string>
  body?: unknown
  params?: Record<string, string | number | boolean>
  timeoutMs?: number
}

export interface APIClient {
  request<T = unknown>(options: APIRequestOptions): Promise<T>
}

export interface CacheManager {
  get<T = unknown>(key: string): T | undefined
  set<T = unknown>(key: string, value: T, ttlMs?: number): void
  remove?(key: string): void
  clear?(): void
}

export interface MessageMetadata extends Record<string, unknown> {
  model?: string
  dialect?: DatabaseDialect | string
  duration?: number
  provider?: string
  raw?: string
}

export interface AIConfig {
  provider: 'ollama' | 'local' | 'openai' | 'deepseek'
  endpoint: string
  model: string
  apiKey: string
  timeout: number
  maxTokens: number
  status: 'connected' | 'disconnected' | 'connecting'
  databaseDialect: DatabaseDialect
}

/**
 * AI Model
 */
export interface Model {
  id: string
  name: string
  size?: string
  description?: string
  maxTokens?: number
}

/**
 * Message in chat
 */
export interface Message {
  id: string
  type: 'user' | 'ai' | 'error'
  content: string
  sql?: string
  meta?: MessageMetadata
  timestamp: number
}

/**
 * Query request
 */
export interface QueryRequest {
  model: string
  prompt: string
  provider: string
  endpoint: string
  apiKey: string
  timeout: number
  maxTokens: number
  includeExplanation: boolean
  databaseDialect: DatabaseDialect
}

/**
 * Query response
 */
export interface QueryResponse {
  success: boolean
  sql?: string
  explanation?: string
  meta?: MessageMetadata
  error?: string
}
