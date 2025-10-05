import type { AIConfig, Model, QueryRequest, QueryResponse } from '@/types'

const API_BASE = '/api/v1/data/query'
const API_STORE = 'ai'

/**
 * AI Service Layer
 * Centralized API calls for AI functionality
 */
export const aiService = {
  /**
   * Fetch available models for a provider
   */
  async fetchModels(provider: string): Promise<Model[]> {
    const result = await callAPI<{ models: Model[] }>('models', { provider })
    return result.models || []
  },

  /**
   * Test connection to AI provider
   */
  async testConnection(config: AIConfig): Promise<boolean> {
    const result = await callAPI<{ success: string }>('test_connection', config)
    return result.success === 'true'
  },

  /**
   * Generate SQL from natural language query
   */
  async generateSQL(request: QueryRequest): Promise<QueryResponse> {
    const result = await callAPI<{
      content: string
      meta: string
      success: string
      error?: string
    }>('generate', {
      model: request.model,
      prompt: request.prompt,
      config: JSON.stringify({
        include_explanation: request.includeExplanation,
        provider: request.provider,
        endpoint: request.endpoint,
        api_key: request.apiKey,
        temperature: request.temperature,
        max_tokens: request.maxTokens
      })
    })

    return {
      success: result.success === 'true',
      sql: result.content,
      meta: result.meta ? JSON.parse(result.meta) : undefined,
      error: result.error
    }
  },

  /**
   * Save AI configuration
   */
  async saveConfig(config: AIConfig): Promise<void> {
    await callAPI('update_config', {
      provider: config.provider,
      config: {
        provider: config.provider,
        endpoint: config.endpoint,
        model: config.model,
        api_key: config.apiKey,
        temperature: config.temperature,
        max_tokens: config.maxTokens
      }
    })
  }
}

/**
 * Call backend API directly
 *
 * @private
 * Note: We use fetch directly instead of DataQuery because DataQuery
 * is designed for database queries and transforms the request format.
 * The AI plugin expects: {type: 'ai', key: 'operation', sql: 'params_json'}
 */
async function callAPI<T>(key: string, data: any): Promise<T> {
  const response = await fetch(API_BASE, {
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
    throw new Error(`API error: ${response.status} ${response.statusText}`)
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
