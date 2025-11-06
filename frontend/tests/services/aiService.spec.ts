import { beforeAll, beforeEach, afterAll, describe, expect, it, vi } from 'vitest'
import { aiService } from '@/services/aiService'

type FetchArgs = Parameters<typeof fetch>

type FetchResponse = ReturnType<typeof fetch>

function createFetchResponse(data: any): FetchResponse {
  return Promise.resolve({
    ok: true,
    status: 200,
    statusText: 'OK',
    headers: {
      get: () => 'application/json'
    },
    json: () => Promise.resolve(data)
  } as Response)
}

describe('aiService', () => {
  const fetchMock = vi.fn<typeof fetch>()

  beforeAll(() => {
    vi.stubGlobal('fetch', fetchMock)
  })

  afterAll(() => {
    vi.unstubAllGlobals()
  })

  beforeEach(() => {
    fetchMock.mockReset()
  })

  it('parses successful SQL generation response with boolean success', async () => {
    fetchMock.mockImplementationOnce(async (_url: FetchArgs[0], options: FetchArgs[1]) => {
      const body = JSON.parse(String(options?.body))
      expect(body).toMatchObject({
        type: 'ai',
        key: 'generate'
      })
      const payload = JSON.parse(body.sql)
      expect(payload.config).toContain('timeout')

      return createFetchResponse({
        data: [
          { key: 'success', value: true },
          { key: 'content', value: 'sql:SELECT 1;\nexplanation:Test query' },
          { key: 'meta', value: '{"confidence":0.9,"model":"demo"}' }
        ]
      })
    })

    const response = await aiService.generateSQL({
      provider: 'ollama',
      endpoint: 'http://localhost:11434',
      apiKey: '',
      model: 'demo',
      prompt: 'Select data',
      timeout: 120,
      maxTokens: 256,
      includeExplanation: true
    })

    expect(response.success).toBe(true)
    expect(response.sql).toBe('SELECT 1;')
    expect(response.explanation).toBe('Test query')
    expect(response.meta).toEqual({ confidence: 0.9, model: 'demo' })
  })

  it('parses health check response when backend returns boolean healthy flag', async () => {
    fetchMock.mockResolvedValueOnce(
      createFetchResponse({
        data: [
          { key: 'healthy', value: true },
          { key: 'provider', value: 'ollama' },
          { key: 'error', value: '' },
          { key: 'timestamp', value: '2025-01-01T00:00:00Z' }
        ]
      })
    )

    const health = await aiService.checkHealth('ollama', 5)
    expect(health.healthy).toBe(true)
    expect(health.provider).toBe('ollama')
  })

  it('fetchModelCatalog should normalize provider keys', async () => {
    const catalogPayload = {
      OpenAI: {
        display_name: 'OpenAI',
        category: 'cloud',
        endpoint: 'https://api.openai.com',
        requires_api_key: true,
        models: [
          { id: 'gpt-5', name: 'GPT-5', description: 'Flagship', max_tokens: 200000 }
        ]
      }
    }

    fetchMock.mockResolvedValueOnce(
      createFetchResponse({
        data: [
          { key: 'catalog', value: JSON.stringify(catalogPayload) },
          { key: 'success', value: true }
        ]
      })
    )

    const catalog = await aiService.fetchModelCatalog()
    expect(catalog.openai).toBeDefined()
    expect(catalog.openai.models).toHaveLength(1)
    expect(catalog.openai.models[0].id).toBe('gpt-5')
  })
})
