import { describe, it, expect, beforeEach, vi } from 'vitest'
import { useAIChat } from '@/composables/useAIChat'
import { aiService } from '@/services/aiService'
import type { AppContext } from '@/types'

// Mock aiService
vi.mock('@/services/aiService', () => ({
  aiService: {
    fetchModels: vi.fn(),
    testConnection: vi.fn(),
    generateSQL: vi.fn(),
    saveConfig: vi.fn()
  }
}))

describe('useAIChat', () => {
  let mockContext: AppContext

  beforeEach(() => {
    localStorage.clear()
    vi.clearAllMocks()

    // Set default mock behavior for fetchModels (called during initialization)
    vi.mocked(aiService.fetchModels).mockResolvedValue([])

    mockContext = {
      i18n: {
        t: (key: string) => key,
        locale: { value: 'en' } as any
      },
      API: {},
      Cache: {}
    }
  })

  describe('initialization', () => {
    it('should initialize with default config', () => {
      const { config, isConfigured, messages, isLoading } = useAIChat(mockContext)

      expect(config.value.provider).toBe('ollama')
      expect(config.value.endpoint).toBe('http://localhost:11434')
      expect(config.value.timeout).toBe(120)
      expect(isConfigured.value).toBe(false)
      expect(messages.value).toEqual([])
      expect(isLoading.value).toBe(false)
    })

    it('should be configured when provider is ollama with endpoint and model', () => {
      localStorage.setItem('atest-ai-config-ollama', JSON.stringify({
        endpoint: 'http://localhost:11434',
        model: 'llama3.2:3b',
        timeout: 150,
        maxTokens: 2048,
        apiKey: '',
        status: 'disconnected'
      }))

      const { isConfigured } = useAIChat(mockContext)
      expect(isConfigured.value).toBe(true)
    })

    it('should be configured when provider is openai with apiKey and model', () => {
      localStorage.setItem('atest-ai-global-config', JSON.stringify({
        provider: 'openai'
      }))
      localStorage.setItem('atest-ai-config-openai', JSON.stringify({
        endpoint: 'https://api.openai.com/v1',
        model: 'gpt-4o',
        timeout: 200,
        maxTokens: 2048,
        apiKey: 'sk-test123',
        status: 'disconnected'
      }))

      const { isConfigured } = useAIChat(mockContext)
      expect(isConfigured.value).toBe(true)
    })
  })

  describe('handleQuery', () => {
    it('should add user message and handle successful response', async () => {
      const mockResponse = {
        success: true,
        sql: 'SELECT * FROM users',
        meta: { model: 'llama3.2:3b', duration: 150 }
      }

      vi.mocked(aiService.generateSQL).mockResolvedValue(mockResponse)

      const { config, messages, handleQuery } = useAIChat(mockContext)

      await handleQuery('show all users', { includeExplanation: false })

      expect(messages.value).toHaveLength(2)
      expect(messages.value[0].type).toBe('user')
      expect(messages.value[0].content).toBe('show all users')
      expect(messages.value[1].type).toBe('ai')
      expect(messages.value[1].sql).toBe('SELECT * FROM users')
      expect(aiService.generateSQL).toHaveBeenCalledWith(expect.objectContaining({
        timeout: config.value.timeout
      }))
    })

    it('should add error message when API fails', async () => {
      vi.mocked(aiService.generateSQL).mockRejectedValue(new Error('API error: 500'))

      const { messages, handleQuery } = useAIChat(mockContext)

      await handleQuery('show all users', { includeExplanation: false })

      expect(messages.value).toHaveLength(2)
      expect(messages.value[0].type).toBe('user')
      expect(messages.value[1].type).toBe('error')
    })

    it('should set loading state correctly', async () => {
      const mockResponse = {
        success: true,
        sql: 'SELECT * FROM users'
      }

      let resolvePromise: any
      vi.mocked(aiService.generateSQL).mockReturnValue(
        new Promise((resolve) => {
          resolvePromise = resolve
        })
      )

      const { isLoading, handleQuery } = useAIChat(mockContext)

      const queryPromise = handleQuery('test', { includeExplanation: false })

      expect(isLoading.value).toBe(true)

      resolvePromise(mockResponse)

      await queryPromise
      expect(isLoading.value).toBe(false)
    })
  })

  describe('handleTestConnection', () => {
    it('should set status to connected on success', async () => {
      vi.mocked(aiService.testConnection).mockResolvedValue({
        success: true,
        message: 'ok',
        provider: 'ollama'
      })

      const { config, handleTestConnection } = useAIChat(mockContext)

      const result = await handleTestConnection()

      expect(result.success).toBe(true)
      expect(config.value.status).toBe('connected')
    })

    it('should set status to disconnected on failure', async () => {
      vi.mocked(aiService.testConnection).mockRejectedValue(new Error('Network error'))

      const { config, handleTestConnection } = useAIChat(mockContext)

      const result = await handleTestConnection()

      expect(result.success).toBe(false)
      expect(result.error).toBe('Network error')
      expect(config.value.status).toBe('disconnected')
    })
  })

  describe('refreshModels', () => {
    it('should fetch and set available models', async () => {
      const mockModels = [
        { id: 'model1', name: 'Model 1', size: '2GB' },
        { id: 'model2', name: 'Model 2', size: '5GB' }
      ]

      vi.mocked(aiService.fetchModels).mockResolvedValue(mockModels)

      const { availableModels, refreshModels } = useAIChat(mockContext)

      await refreshModels()

      expect(availableModels.value).toHaveLength(2)
      expect(availableModels.value[0].id).toBe('model1')
    })

    it('should use mock models when API fails', async () => {
      vi.mocked(aiService.fetchModels).mockRejectedValue(new Error('Network error'))

      const { availableModels, refreshModels } = useAIChat(mockContext)

      await refreshModels()

      expect(availableModels.value.length).toBeGreaterThan(0)
      expect(availableModels.value[0].id).toBe('llama3.2:3b')
    })

    it('should auto-select first model if none selected', async () => {
      const mockModels = [
        { id: 'auto-model', name: 'Auto Model', size: '1GB' }
      ]

      vi.mocked(aiService.fetchModels).mockResolvedValue(mockModels)

      const { config, refreshModels } = useAIChat(mockContext)

      expect(config.value.model).toBe('')

      await refreshModels()

      expect(config.value.model).toBe('auto-model')
    })
  })

  describe('config persistence', () => {
    it('should save config to localStorage when changed', async () => {
      const { config } = useAIChat(mockContext)

      config.value.model = 'new-model'
      config.value.maxTokens = 4096

      // Wait for watch to trigger
      await new Promise(resolve => setTimeout(resolve, 10))

      const saved = JSON.parse(localStorage.getItem('atest-ai-config-ollama')!)
      expect(saved.model).toBe('new-model')
      expect(saved.maxTokens).toBe(4096)
    })
  })
})
