import { describe, it, expect, beforeEach } from 'vitest'
import {
  loadConfig,
  saveConfig,
  getDefaultConfig,
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
      expect(config.timeout).toBe(120)
      expect(config.maxTokens).toBe(2048)
      expect(config.status).toBe('disconnected')
    })

    it('should load saved provider from global config', () => {
      localStorage.setItem('atest-ai-global-config', JSON.stringify({
        provider: 'openai'
      }))

      const config = loadConfig()
      expect(config.provider).toBe('openai')
      expect(config.endpoint).toBe('https://api.openai.com')
      expect(config.apiKey).toBe('')
      expect(config.timeout).toBe(120)
    })

    it('should load provider-specific config', () => {
      localStorage.setItem('atest-ai-global-config', JSON.stringify({
        provider: 'ollama'
      }))
      localStorage.setItem('atest-ai-config-ollama', JSON.stringify({
        endpoint: 'http://localhost:11434',
        model: 'llama3.2:3b',
        timeout: 180,
        maxTokens: 1024,
        apiKey: '',
        status: 'connected'
      }))

      const config = loadConfig()
      expect(config.provider).toBe('ollama')
      expect(config.model).toBe('llama3.2:3b')
      expect(config.timeout).toBe(180)
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
        maxTokens: 1024,
        timeout: 90
      }))

      const config = loadConfig()
      expect(config.provider).toBe('deepseek')
      expect(config.endpoint).toBe('https://api.deepseek.com')
      expect(config.model).toBe('deepseek-chat')
      expect(config.timeout).toBe(90)
    })

    it('should normalize trailing version segment for openai endpoint', () => {
      localStorage.setItem('atest-ai-global-config', JSON.stringify({
        provider: 'openai'
      }))
      localStorage.setItem('atest-ai-config-openai', JSON.stringify({
        endpoint: 'https://api.openai.com/v1/',
        model: 'gpt-5',
        apiKey: 'sk-test'
      }))

      const config = loadConfig()
      expect(config.endpoint).toBe('https://api.openai.com')
    })
  })

  describe('saveConfig', () => {
    it('should save config to localStorage', () => {
      const config: AIConfig = {
        provider: 'deepseek',
        endpoint: 'https://api.deepseek.com',
        model: 'deepseek-chat',
        apiKey: 'sk-test123',
        timeout: 240,
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
      expect(providerConfig.timeout).toBe(240)
      expect(providerConfig.provider).toBeUndefined()
      expect(providerConfig.status).toBeUndefined()
    })

    it('should normalize local provider key to ollama', () => {
      const config: AIConfig = {
        provider: 'local',
        endpoint: 'http://localhost:11434',
        model: 'llama3.2:3b',
        apiKey: '',
        timeout: 90,
        maxTokens: 1024,
        status: 'connected'
      }

      saveConfig(config)

      const providerConfig = JSON.parse(localStorage.getItem('atest-ai-config-ollama')!)
      expect(providerConfig.endpoint).toBe('http://localhost:11434')
      expect(providerConfig.model).toBe('llama3.2:3b')
      expect(providerConfig.timeout).toBe(90)
    })

    it('should normalize openai endpoint when saving', () => {
      const config: AIConfig = {
        provider: 'openai',
        endpoint: 'https://api.openai.com/v1',
        model: 'gpt-5',
        apiKey: 'sk-test',
        timeout: 120,
        maxTokens: 16384,
        status: 'connected'
      }

      saveConfig(config)

      const providerConfig = JSON.parse(localStorage.getItem('atest-ai-config-openai')!)
      expect(providerConfig.endpoint).toBe('https://api.openai.com')
    })
  })

  describe('getDefaultConfig', () => {
    it('should return ollama default config', () => {
      const config = getDefaultConfig('ollama')

      expect(config.endpoint).toBe('http://localhost:11434')
      expect(config.apiKey).toBe('')
      expect(config.timeout).toBe(120)
      expect(config.maxTokens).toBe(2048)
    })

    it('should return openai default config', () => {
      const config = getDefaultConfig('openai')

      expect(config.endpoint).toBe('https://api.openai.com')
      expect(config.apiKey).toBe('')
      expect(config.timeout).toBe(120)
    })

    it('should return ollama config for unknown provider', () => {
      const config = getDefaultConfig('unknown')
      expect(config.endpoint).toBe('http://localhost:11434')
      expect(config.timeout).toBe(120)
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
