import { describe, it, expect, beforeEach } from 'vitest'
import {
  loadConfig,
  saveConfig,
  getDefaultConfig,
  getMockModels,
  generateId
} from '@/utils/config'
import type { AIConfig } from '@/types'

describe('config utils', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  describe('loadConfig', () => {
    it('should load default config when no config exists', () => {
      const config = loadConfig()

      expect(config.provider).toBe('ollama')
      expect(config.endpoint).toBe('http://localhost:11434')
      expect(config.model).toBe('')
      expect(config.maxTokens).toBe(2048)
      expect(config.status).toBe('disconnected')
    })

    it('should load saved provider from global config', () => {
      localStorage.setItem('atest-ai-global-config', JSON.stringify({
        provider: 'openai'
      }))

      const config = loadConfig()
      expect(config.provider).toBe('openai')
      expect(config.endpoint).toBe('https://api.openai.com/v1')
      expect(config.apiKey).toBe('')
    })

    it('should load provider-specific config', () => {
      localStorage.setItem('atest-ai-global-config', JSON.stringify({
        provider: 'ollama'
      }))
      localStorage.setItem('atest-ai-config-ollama', JSON.stringify({
        endpoint: 'http://localhost:11434',
        model: 'llama3.2:3b',
        maxTokens: 1024,
        apiKey: '',
        status: 'connected'
      }))

      const config = loadConfig()
      expect(config.provider).toBe('ollama')
      expect(config.model).toBe('llama3.2:3b')
      expect(config.maxTokens).toBe(1024)
      expect(config.status).toBe('connected')
    })

    it('should sanitize local endpoint when provider is deepseek', () => {
      localStorage.setItem('atest-ai-global-config', JSON.stringify({
        provider: 'deepseek'
      }))
      localStorage.setItem('atest-ai-config-deepseek', JSON.stringify({
        endpoint: 'http://localhost:11434',
        model: 'deepseek-chat',
        apiKey: 'sk-test',
        maxTokens: 1024
      }))

      const config = loadConfig()
      expect(config.provider).toBe('deepseek')
      expect(config.endpoint).toBe('https://api.deepseek.com')
      expect(config.model).toBe('deepseek-chat')
    })
  })

  describe('saveConfig', () => {
    it('should save config to localStorage', () => {
      const config: AIConfig = {
        provider: 'deepseek',
        endpoint: 'https://api.deepseek.com',
        model: 'deepseek-chat',
        apiKey: 'sk-test123',
        maxTokens: 2048,
        status: 'disconnected'
      }

      saveConfig(config)

      const globalConfig = JSON.parse(localStorage.getItem('atest-ai-global-config')!)
      expect(globalConfig.provider).toBe('deepseek')

      const providerConfig = JSON.parse(localStorage.getItem('atest-ai-config-deepseek')!)
      expect(providerConfig.endpoint).toBe('https://api.deepseek.com')
      expect(providerConfig.model).toBe('deepseek-chat')
      expect(providerConfig.apiKey).toBe('sk-test123')
      expect(providerConfig.provider).toBeUndefined()
      expect(providerConfig.status).toBeUndefined()
    })

    it('should normalize local provider key to ollama', () => {
      const config: AIConfig = {
        provider: 'local',
        endpoint: 'http://localhost:11434',
        model: 'llama3.2:3b',
        apiKey: '',
        maxTokens: 1024,
        status: 'connected'
      }

      saveConfig(config)

      const providerConfig = JSON.parse(localStorage.getItem('atest-ai-config-ollama')!)
      expect(providerConfig.endpoint).toBe('http://localhost:11434')
      expect(providerConfig.model).toBe('llama3.2:3b')
    })
  })

  describe('getDefaultConfig', () => {
    it('should return ollama default config', () => {
      const config = getDefaultConfig('ollama')

      expect(config.endpoint).toBe('http://localhost:11434')
      expect(config.apiKey).toBe('')
      expect(config.maxTokens).toBe(2048)
    })

    it('should return openai default config', () => {
      const config = getDefaultConfig('openai')

      expect(config.endpoint).toBe('https://api.openai.com/v1')
      expect(config.apiKey).toBe('')
    })

    it('should return ollama config for unknown provider', () => {
      const config = getDefaultConfig('unknown')
      expect(config.endpoint).toBe('http://localhost:11434')
    })
  })

  describe('getMockModels', () => {
    it('should return ollama mock models', () => {
      const models = getMockModels('ollama')

      expect(models).toHaveLength(2)
      expect(models[0].id).toBe('llama3.2:3b')
      expect(models[0].name).toBe('Llama 3.2 3B')
    })

    it('should return openai mock models', () => {
      const models = getMockModels('openai')

      expect(models.length).toBeGreaterThanOrEqual(7)
      expect(models[0].id).toBe('gpt-5')
      expect(models.some(model => model.id === 'gpt-4o-2024-08-06')).toBe(true)
    })

    it('should return empty array for unknown provider', () => {
      const models = getMockModels('unknown')
      expect(models).toEqual([])
    })
  })

  describe('generateId', () => {
    it('should generate unique IDs', () => {
      const id1 = generateId()
      const id2 = generateId()

      expect(id1).not.toBe(id2)
      expect(id1).toMatch(/^\d+-[a-z0-9]+$/)
    })
  })
})
