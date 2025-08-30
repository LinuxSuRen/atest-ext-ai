import { defineStore } from 'pinia'
import { ref, computed, readonly } from 'vue'
import { apiService, type ConvertToSQLRequest, type ConvertToSQLResponse } from '@/services/api'
import { useSettingsStore } from './settings'

export interface QueryHistory {
  id: string
  query: string
  sql: string
  explanation?: string
  timestamp: Date
  confidence?: number
}

export const useAIStore = defineStore('ai', () => {
  // 状态
  const isLoading = ref(false)
  const currentQuery = ref('')
  const currentSQL = ref('')
  const currentExplanation = ref('')
  const queryHistory = ref<QueryHistory[]>([])
  const error = ref<string | null>(null)
  const modelInfo = ref<any>(null)
  const isConnected = ref(false)

  // 计算属性
  const hasHistory = computed(() => queryHistory.value.length > 0)
  const latestQuery = computed(() => queryHistory.value[0] || null)

  // Actions
  const convertToSQL = async (query: string): Promise<void> => {
    if (!query.trim()) {
      error.value = '请输入查询内容'
      return
    }

    isLoading.value = true
    error.value = null
    currentQuery.value = query

    try {
      const response = await apiService.convertToSQL({
        query,
        context: JSON.stringify({
          database_type: 'postgresql',
          schema_info: [],
        }),
      })

      currentSQL.value = response.sql
      currentExplanation.value = response.explanation || ''
      
      // 添加到历史记录
      const historyItem: QueryHistory = {
        id: Date.now().toString(),
        query,
        sql: response.sql,
        explanation: response.explanation,
        confidence: response.confidence,
        timestamp: new Date()
      }
      
      queryHistory.value.unshift(historyItem)
      
      // 限制历史记录数量
      const settingsStore = useSettingsStore()
      const maxHistory = settingsStore.historyLimit
      if (queryHistory.value.length > maxHistory) {
        queryHistory.value = queryHistory.value.slice(0, maxHistory)
      }
      
      // 保存到本地存储
      if (settingsStore.autoSave) {
        saveToLocalStorage()
      }
    } catch (err) {
      error.value = err instanceof Error ? err.message : '转换失败'
      console.error('SQL conversion error:', err)
    } finally {
      isLoading.value = false
    }
  }

  const saveToLocalStorage = () => {
    try {
      localStorage.setItem('ai-query-history', JSON.stringify(queryHistory.value))
    } catch (err) {
      console.error('Failed to save history to localStorage:', err)
    }
  }

  const loadFromLocalStorage = () => {
    try {
      const saved = localStorage.getItem('ai-query-history')
      if (saved) {
        const parsed = JSON.parse(saved)
        queryHistory.value = parsed.map((item: any) => ({
          ...item,
          timestamp: new Date(item.timestamp)
        }))
      }
    } catch (err) {
      console.error('Failed to load history from localStorage:', err)
    }
  }

  const clearHistory = (): void => {
    queryHistory.value = []
    saveToLocalStorage()
  }

  const removeHistoryItem = (id: string): void => {
    const index = queryHistory.value.findIndex(item => item.id === id)
    if (index > -1) {
      queryHistory.value.splice(index, 1)
      saveToLocalStorage()
    }
  }

  const checkHealth = async (): Promise<boolean> => {
    try {
      await apiService.healthCheck()
      return true
    } catch {
      return false
    }
  }

  const loadModelInfo = async (): Promise<void> => {
    try {
      const info = await apiService.getModelInfo()
      console.log('Model info loaded:', info)
    } catch (err) {
      console.error('Failed to load model info:', err)
    }
  }

  const clearError = () => {
    error.value = null
  }

  const clearCurrentResult = () => {
    currentQuery.value = ''
    currentSQL.value = ''
    currentExplanation.value = ''
  }

  // 初始化时加载历史记录
  loadFromLocalStorage()

  return {
    // State
    isLoading: readonly(isLoading),
    currentQuery: readonly(currentQuery),
    currentSQL: readonly(currentSQL),
    currentExplanation: readonly(currentExplanation),
    queryHistory: readonly(queryHistory),
    error: readonly(error),
    modelInfo: readonly(modelInfo),
    isConnected: readonly(isConnected),
    
    // Computed
    hasHistory,
    latestQuery,
    
    // Actions
    convertToSQL,
    clearHistory,
    removeHistoryItem,
    checkHealth,
    loadModelInfo,
    clearError,
    clearCurrentResult,
    saveToLocalStorage,
    loadFromLocalStorage
  }
})