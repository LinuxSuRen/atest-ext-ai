/**
 * AI连接失败时的降级处理机制
 */

export interface FallbackConfig {
  maxRetries: number
  retryDelay: number
  timeoutMs: number
  enableFallback: boolean
}

export interface ErrorInfo {
  type: 'network' | 'timeout' | 'server' | 'auth' | 'unknown'
  message: string
  code?: string | number
  timestamp: number
}

export interface FallbackSuggestion {
  type: 'manual' | 'template' | 'retry' | 'offline'
  title: string
  description: string
  action?: () => void
}

/**
 * 检测错误类型
 */
export function detectErrorType(error: any): ErrorInfo['type'] {
  if (!error) return 'unknown'
  
  const message = error.message || error.toString().toLowerCase()
  const code = (error as any).code || (error as any).status
  
  // 网络错误
  if (message.includes('network') || message.includes('fetch') || code === 'NETWORK_ERROR') {
    return 'network'
  }
  
  // 超时错误
  if (message.includes('timeout') || code === 'TIMEOUT' || code === 408) {
    return 'timeout'
  }
  
  // 认证错误
  if (code === 401 || code === 403 || message.includes('unauthorized') || message.includes('forbidden')) {
    return 'auth'
  }
  
  // 服务器错误
  if (code >= 500 || message.includes('server') || message.includes('internal')) {
    return 'server'
  }
  
  return 'unknown'
}

/**
 * 错误处理和降级管理器
 */
class FallbackManager {
  private config: FallbackConfig = {
    maxRetries: 3,
    retryDelay: 1000,
    timeoutMs: 30000,
    enableFallback: true
  }
  
  private retryCount = 0
  private lastError: ErrorInfo | null = null
  private isInFallbackMode = false
  
  /**
   * 设置配置
   */
  setConfig(config: Partial<FallbackConfig>) {
    this.config = { ...this.config, ...config }
  }
  
  /**
   * 处理AI请求错误
   */
  async handleError(error: Error): Promise<{ shouldRetry: boolean; suggestion?: FallbackSuggestion }> {
    const errorType = detectErrorType(error)
    
    this.lastError = {
      type: errorType,
      message: error.message || '未知错误',
      code: (error as any).code || (error as any).status,
      timestamp: Date.now()
    }
    
    console.error('AI请求错误:', this.lastError)
    
    // 检查是否应该重试
    const shouldRetry = this.shouldRetry(errorType)
    
    if (shouldRetry) {
      this.retryCount++
      await this.delay(this.config.retryDelay * this.retryCount)
      return { shouldRetry: true }
    }
    
    // 进入降级模式
    this.isInFallbackMode = true
    const suggestion = this.generateFallbackSuggestion(errorType)
    
    return { shouldRetry: false, suggestion }
  }
  
  /**
   * 判断是否应该重试
   */
  private shouldRetry(errorType: ErrorInfo['type']): boolean {
    if (this.retryCount >= this.config.maxRetries) {
      return false
    }
    
    // 网络和超时错误可以重试
    return errorType === 'network' || errorType === 'timeout'
  }
  
  /**
   * 生成降级建议
   */
  private generateFallbackSuggestion(errorType: ErrorInfo['type']): FallbackSuggestion {
    switch (errorType) {
      case 'network':
        return {
          type: 'retry',
          title: '网络连接异常',
          description: '请检查网络连接后重试，或手动编写SQL查询',
          action: () => this.reset()
        }
      
      case 'timeout':
        return {
          type: 'retry',
          title: '请求超时',
          description: '服务响应较慢，请稍后重试或使用离线模式',
          action: () => this.reset()
        }
      
      case 'auth':
        return {
          type: 'manual',
          title: '认证失败',
          description: '请检查API密钥配置，或使用手动SQL编写模式'
        }
      
      case 'server':
        return {
          type: 'offline',
          title: '服务暂时不可用',
          description: 'AI服务正在维护中，建议使用离线模式或稍后重试'
        }
      
      default:
        return {
          type: 'manual',
          title: '未知错误',
          description: '遇到未知问题，建议手动编写SQL查询或联系技术支持'
        }
    }
  }
  
  /**
   * 延迟函数
   */
  private delay(ms: number): Promise<void> {
    return new Promise(resolve => setTimeout(resolve, ms))
  }
  
  /**
   * 重置状态
   */
  reset() {
    this.retryCount = 0
    this.lastError = null
    this.isInFallbackMode = false
  }
  
  /**
   * 获取当前状态
   */
  getStatus() {
    return {
      isInFallbackMode: this.isInFallbackMode,
      retryCount: this.retryCount,
      lastError: this.lastError,
      canRetry: this.retryCount < this.config.maxRetries
    }
  }
  
  /**
   * 生成SQL模板建议
   */
  generateSQLTemplates(): Array<{ title: string; sql: string; description: string }> {
    return [
      {
        title: '查询所有数据',
        sql: 'SELECT * FROM table_name LIMIT 10;',
        description: '查询表中的前10条记录'
      },
      {
        title: '条件查询',
        sql: 'SELECT * FROM table_name WHERE column_name = \'value\';',
        description: '根据条件查询特定数据'
      },
      {
        title: '统计查询',
        sql: 'SELECT COUNT(*) FROM table_name;',
        description: '统计表中的记录总数'
      },
      {
        title: '分组统计',
        sql: 'SELECT column_name, COUNT(*) FROM table_name GROUP BY column_name;',
        description: '按字段分组统计'
      },
      {
        title: '排序查询',
        sql: 'SELECT * FROM table_name ORDER BY column_name DESC LIMIT 10;',
        description: '按字段排序查询'
      }
    ]
  }
}

// 创建全局实例
export const fallbackManager = new FallbackManager()

/**
 * Vue 3 Composition API Hook
 */
import { ref, computed } from 'vue'

export function useFallback() {
  const status = ref(fallbackManager.getStatus())
  
  const handleError = async (error: Error) => {
    const result = await fallbackManager.handleError(error)
    status.value = fallbackManager.getStatus()
    return result
  }
  
  const reset = () => {
    fallbackManager.reset()
    status.value = fallbackManager.getStatus()
  }
  
  const getSQLTemplates = () => {
    return fallbackManager.generateSQLTemplates()
  }
  
  return {
    status: computed(() => status.value),
    isInFallbackMode: computed(() => status.value.isInFallbackMode),
    canRetry: computed(() => status.value.canRetry),
    lastError: computed(() => status.value.lastError),
    handleError,
    reset,
    getSQLTemplates
  }
}