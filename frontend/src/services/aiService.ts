import type { AIConfig, Model, QueryRequest, QueryResponse } from '@/types'

const API_BASE = '/api/v1/data/query'
const API_STORE = 'ai'

function toBoolean(value: unknown): boolean {
  if (typeof value === 'boolean') {
    return value
  }
  if (typeof value === 'string') {
    const normalized = value.trim().toLowerCase()
    if (normalized === 'true') {
      return true
    }
    if (normalized === 'false') {
      return false
    }
  }
  return Boolean(value)
}

function safeParseJSON<T>(value: unknown): T | undefined {
  if (typeof value !== 'string') {
    return value as T | undefined
  }
  try {
    return JSON.parse(value) as T
  } catch (error) {
    console.warn('[aiService] Failed to parse JSON value from backend', value, error)
    return value as T | undefined
  }
}

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
  async testConnection(config: AIConfig): Promise<{
    success: boolean
    message: string
    provider: string
    error?: string
  }> {
    const result = await callAPI<{
      success: string | boolean
      message: string
      provider: string
      error?: string
    }>('test_connection', config)

    return {
      success: toBoolean(result.success),
      message: result.message || '',
      provider: result.provider || config.provider,
      error: result.error
    }
  },

  /**
   * Check AI service health (does not affect plugin Ready status)
   */
  async checkHealth(provider: string = '', timeout: number = 5): Promise<{
    healthy: boolean
    provider: string
    error: string
    timestamp: string
  }> {
    const result = await callAPI<{
      healthy: string | boolean
      provider: string
      error: string
      timestamp: string
    }>('health_check', {
      provider,
      timeout
    })

    return {
      healthy: toBoolean(result.healthy),
      provider: result.provider,
      error: result.error || '',
      timestamp: result.timestamp
    }
  },

  /**
   * Generate SQL from natural language query
   */
  async generateSQL(request: QueryRequest): Promise<QueryResponse> {
    console.log('📤 [aiService] generateSQL called', {
      model: request.model,
      provider: request.provider,
      endpoint: request.endpoint,
      promptLength: request.prompt.length,
      includeExplanation: request.includeExplanation
    })

    try {
      const result = await callAPI<{
        content: string
        meta: string
        success: string | boolean
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

      console.log('📥 [aiService] Received backend result', {
        hasContent: !!result.content,
        contentLength: result.content?.length || 0,
        success: result.success,
        hasError: !!result.error,
        hasMeta: !!result.meta
      })

      // Parse backend format: "sql:xxx\nexplanation:xxx"
      let sql = ''
      let explanation = ''

      if (result.content) {
        const lines = result.content.split('\n')
        for (const line of lines) {
          if (line.startsWith('sql:')) {
            sql = line.substring(4).trim()
          } else if (line.startsWith('explanation:')) {
            explanation = line.substring(12).trim()
          }
        }
      }

      const response = {
        success: toBoolean(result.success),
        sql,
        explanation: explanation || undefined,
        meta: safeParseJSON(result.meta) ?? result.meta,
        error: result.error
      }

      console.log('✅ [aiService] Parsed response', {
        success: response.success,
        hasSql: !!response.sql,
        sqlLength: response.sql?.length || 0,
        hasExplanation: !!response.explanation,
        hasError: !!response.error
      })

      return response
    } catch (error) {
      console.error('❌ [aiService] generateSQL failed', {
        error,
        message: (error as Error).message,
        stack: (error as Error).stack
      })
      throw error
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
  const requestBody = {
    type: 'ai',
    key,
    sql: JSON.stringify(data)
  }

  console.log('🌐 [callAPI] Sending request', {
    url: API_BASE,
    key,
    dataKeys: Object.keys(data),
    bodyLength: JSON.stringify(requestBody).length
  })

  try {
    const response = await fetch(API_BASE, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Store-Name': API_STORE
      },
      body: JSON.stringify(requestBody)
    })

    console.log('📡 [callAPI] Received HTTP response', {
      status: response.status,
      statusText: response.statusText,
      ok: response.ok,
      contentType: response.headers.get('content-type')
    })

    if (!response.ok) {
      const errorText = await response.text()
      console.error('❌ [callAPI] HTTP error', {
        status: response.status,
        statusText: response.statusText,
        body: errorText
      })
      throw new Error(`API error: ${response.status} ${response.statusText} - ${errorText}`)
    }

    const result = await response.json()
    console.log('📦 [callAPI] Parsed JSON result', {
      hasData: !!result.data,
      dataLength: result.data?.length || 0,
      resultKeys: Object.keys(result)
    })

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
      console.log('🔓 [callAPI] Parsed data pairs', {
        keys: Object.keys(parsed)
      })
    }

    return parsed as T
  } catch (error) {
    console.error('💥 [callAPI] Request failed', {
      error,
      message: (error as Error).message,
      stack: (error as Error).stack
    })
    throw error
  }
}
