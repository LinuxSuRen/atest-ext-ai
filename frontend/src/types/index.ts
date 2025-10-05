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
  API: any  // Main app's API object from net.ts
  Cache: any  // Main app's Cache object from cache.ts
}

/**
 * AI configuration
 */
export interface AIConfig {
  provider: 'ollama' | 'openai' | 'deepseek'
  language: 'en' | 'zh'
  endpoint: string
  model: string
  apiKey: string
  temperature: number
  maxTokens: number
  status: 'connected' | 'disconnected' | 'connecting'
}

/**
 * AI Model
 */
export interface Model {
  id: string
  name: string
  size: string
}

/**
 * Message in chat
 */
export interface Message {
  id: string
  type: 'user' | 'ai' | 'error'
  content: string
  sql?: string
  meta?: any
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
  temperature: number
  maxTokens: number
  includeExplanation: boolean
}

/**
 * Query response
 */
export interface QueryResponse {
  success: boolean
  sql?: string
  explanation?: string
  meta?: any
  error?: string
}
