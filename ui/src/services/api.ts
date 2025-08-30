import axios, { AxiosError } from 'axios'

// API 基础配置
const API_BASE_URL = '/api'

// 创建 axios 实例
const apiClient = axios.create({
  baseURL: API_BASE_URL,
  timeout: 30000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// 错误类型定义
export interface APIError extends Error {
  code?: string | number
  status?: number
  type?: 'network' | 'timeout' | 'server' | 'auth' | 'unknown'
}

// 创建增强的错误对象
function createAPIError(error: AxiosError | Error, type?: APIError['type']): APIError {
  const apiError = new Error(error.message) as APIError
  apiError.name = 'APIError'
  
  if ('response' in error && error.response) {
    apiError.status = error.response.status
    apiError.code = error.response.status
    
    // 根据HTTP状态码确定错误类型
    if (error.response.status >= 500) {
      apiError.type = 'server'
    } else if (error.response.status === 401 || error.response.status === 403) {
      apiError.type = 'auth'
    } else {
      apiError.type = 'server'
    }
  } else if ('code' in error) {
    apiError.code = error.code
    
    // 根据错误代码确定类型
    if (error.code === 'ECONNABORTED' || error.message.includes('timeout')) {
      apiError.type = 'timeout'
    } else if (error.code === 'NETWORK_ERROR' || error.message.includes('Network Error')) {
      apiError.type = 'network'
    } else {
      apiError.type = type || 'unknown'
    }
  } else {
    apiError.type = type || 'unknown'
  }
  
  return apiError
}

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    // 可以在这里添加认证token等
    console.log('API Request:', config.method?.toUpperCase(), config.url)
    return config
  },
  (error) => {
    console.error('Request Error:', error)
    return Promise.reject(createAPIError(error, 'network'))
  }
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => {
    console.log('API Response:', response.status, response.config.url)
    return response.data as any
  },
  (error: AxiosError) => {
    console.error('API Error:', {
      url: error.config?.url,
      method: error.config?.method,
      status: error.response?.status,
      message: error.message,
      code: error.code
    })
    
    const apiError = createAPIError(error)
    return Promise.reject(apiError)
  }
)

// AI 相关接口
export interface ConvertToSQLRequest {
  query: string
  context?: string
}

export interface ConvertToSQLResponse {
  sql: string
  explanation?: string
  confidence?: number
  success?: boolean
  warnings?: string[]
  model?: string
  provider?: string
}

export interface HealthCheckResponse {
  status: string
  timestamp: string
  healthy?: boolean
}

export interface ModelInfoResponse {
  name: string
  version: string
  capabilities: string[]
  provider?: string
  limits?: Record<string, number>
  metadata?: Record<string, string>
}

// 重试配置
interface RetryConfig {
  maxRetries: number
  retryDelay: number
  retryableErrors: APIError['type'][]
}

const defaultRetryConfig: RetryConfig = {
  maxRetries: 3,
  retryDelay: 1000,
  retryableErrors: ['network', 'timeout']
}

// 重试函数
async function withRetry<T>(
  fn: () => Promise<T>,
  config: RetryConfig = defaultRetryConfig
): Promise<T> {
  let lastError: APIError
  
  for (let attempt = 0; attempt <= config.maxRetries; attempt++) {
    try {
      return await fn()
    } catch (error) {
      lastError = error as APIError
      
      // 如果不是可重试的错误类型，直接抛出
      if (!config.retryableErrors.includes(lastError.type || 'unknown')) {
        throw lastError
      }
      
      // 如果是最后一次尝试，抛出错误
      if (attempt === config.maxRetries) {
        throw lastError
      }
      
      // 等待后重试
      const delay = config.retryDelay * Math.pow(2, attempt) // 指数退避
      console.log(`API请求失败，${delay}ms后进行第${attempt + 1}次重试...`)
      await new Promise(resolve => setTimeout(resolve, delay))
    }
  }
  
  throw lastError!
}

// API 方法
export const aiAPI = {
  // 自然语言转SQL（带重试机制）
  convertToSQL: async (request: ConvertToSQLRequest): Promise<ConvertToSQLResponse> => {
    return withRetry(async () => {
      const response = await apiClient.post('/convert-to-sql', request)
      
      // 验证响应数据
      if (!response || typeof response !== 'object') {
        throw createAPIError(new Error('Invalid response format'), 'server')
      }
      
      if (!(response as any).sql) {
        throw createAPIError(new Error('No SQL generated in response'), 'server')
      }
      
      return response as unknown as ConvertToSQLResponse
    })
  },

  // 健康检查（带重试机制）
  healthCheck: async (): Promise<HealthCheckResponse> => {
    return withRetry(async () => {
      const response = await apiClient.get('/health')
      return response as unknown as HealthCheckResponse
    }, {
      maxRetries: 1, // 健康检查只重试一次
      retryDelay: 500,
      retryableErrors: ['network', 'timeout']
    })
  },

  // 获取模型信息（带重试机制）
  getModelInfo: async (): Promise<ModelInfoResponse> => {
    return withRetry(async () => {
      const response = await apiClient.get('/model-info')
      return response as unknown as ModelInfoResponse
    }, {
      maxRetries: 2,
      retryDelay: 1000,
      retryableErrors: ['network', 'timeout']
    })
  },

  // Ping测试（不重试）
  ping: async (): Promise<{ message: string }> => {
    const response = await apiClient.get('/ping')
    
    // 验证响应格式
    if (!response || typeof response !== 'object') {
      throw createAPIError(new Error('Invalid ping response format'), 'server')
    }
    
    return response as unknown as { message: string }
  },
}

// 导出 apiService 作为默认服务
export const apiService = aiAPI

export default apiClient

// 导出错误创建函数供其他模块使用
export { createAPIError }