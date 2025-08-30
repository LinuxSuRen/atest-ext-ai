import axios from 'axios'

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

// 请求拦截器
apiClient.interceptors.request.use(
  (config) => {
    // 可以在这里添加认证token等
    return config
  },
  (error) => {
    return Promise.reject(error)
  }
)

// 响应拦截器
apiClient.interceptors.response.use(
  (response) => {
    return response.data
  },
  (error) => {
    console.error('API Error:', error)
    return Promise.reject(error)
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
}

export interface HealthCheckResponse {
  status: string
  timestamp: string
}

export interface ModelInfoResponse {
  name: string
  version: string
  capabilities: string[]
}

// API 方法
export const aiAPI = {
  // 自然语言转SQL
  convertToSQL: async (request: ConvertToSQLRequest): Promise<ConvertToSQLResponse> => {
    return apiClient.post('/convert-to-sql', request)
  },

  // 健康检查
  healthCheck: async (): Promise<HealthCheckResponse> => {
    return apiClient.get('/health')
  },

  // 获取模型信息
  getModelInfo: async (): Promise<ModelInfoResponse> => {
    return apiClient.get('/model-info')
  },

  // Ping测试
  ping: async (): Promise<{ message: string }> => {
    return apiClient.get('/ping')
  },
}

// 导出 apiService 作为默认服务
export const apiService = aiAPI

export default apiClient