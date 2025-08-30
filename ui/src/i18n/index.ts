/**
 * 国际化配置
 */

export interface I18nMessages {
  [key: string]: string | I18nMessages
}

export const messages = {
  'zh-CN': {
    ai: {
      title: 'AI SQL 助手',
      placeholder: '请输入您的查询需求...',
      send: '发送',
      clear: '清空',
      copy: '复制',
      execute: '执行',
      edit: '编辑',
      loading: '正在生成SQL...',
      sqlGenerated: 'SQL已生成',
      error: {
        connection: 'AI服务连接失败',
        generation: 'SQL生成失败',
        execution: 'SQL执行失败',
        network: '网络连接异常',
        timeout: '请求超时',
        unknown: '未知错误'
      },
      message: {
        user: '您',
        assistant: 'AI助手',
        copied: 'SQL已复制到剪贴板',
        executed: 'SQL执行成功',
        cleared: '对话已清空'
      },
      status: {
        connected: '已连接',
        disconnected: '未连接',
        connecting: '连接中...',
        error: '连接错误'
      },
      fallback: {
        title: '离线模式',
        description: 'AI服务暂时不可用，您可以手动编写SQL查询',
        suggestion: '您可以尝试使用以下SQL模板'
      }
    },
    welcome: {
      message: '您好！我是AI SQL助手，可以帮您生成和优化SQL查询。请告诉我您的需求。'
    },
    examples: {
      query1: '查询所有用户的基本信息',
      query2: '统计每个部门的员工数量',
      query3: '查找最近30天的订单记录'
    },
    errors: {
      aiRequestFailed: 'AI请求失败，请稍后重试',
      sqlExecutionFailed: 'SQL执行失败',
      aiServiceUnavailable: 'AI服务暂时不可用，请检查网络连接'
    },
    sql: {
      executionSuccess: '执行成功，返回 {rows} 条记录'
    }
  },
  'en-US': {
    ai: {
      title: 'AI SQL Assistant',
      placeholder: 'Enter your query requirements...',
      send: 'Send',
      clear: 'Clear',
      copy: 'Copy',
      execute: 'Execute',
      edit: 'Edit',
      loading: 'Generating SQL...',
      sqlGenerated: 'SQL generated',
      error: {
        connection: 'AI service connection failed',
        generation: 'SQL generation failed',
        execution: 'SQL execution failed',
        network: 'Network connection error',
        timeout: 'Request timeout',
        unknown: 'Unknown error'
      },
      message: {
        user: 'You',
        assistant: 'AI Assistant',
        copied: 'SQL copied to clipboard',
        executed: 'SQL executed successfully',
        cleared: 'Conversation cleared'
      },
      status: {
        connected: 'Connected',
        disconnected: 'Disconnected',
        connecting: 'Connecting...',
        error: 'Connection Error'
      },
      fallback: {
        title: 'Offline Mode',
        description: 'AI service is temporarily unavailable, you can write SQL queries manually',
        suggestion: 'You can try using the following SQL templates'
      }
    },
    welcome: {
      message: 'Hello! I am your AI SQL Assistant. I can help you generate and optimize SQL queries. Please tell me what you need.'
    },
    examples: {
      query1: 'Query all user basic information',
      query2: 'Count employees by department',
      query3: 'Find order records from the last 30 days'
    },
    errors: {
      aiRequestFailed: 'AI request failed, please try again later',
      sqlExecutionFailed: 'SQL execution failed',
      aiServiceUnavailable: 'AI service is temporarily unavailable, please check your network connection'
    },
    sql: {
      executionSuccess: 'Execution successful, returned {rows} records'
    }
  }
}

/**
 * 获取嵌套对象的值
 */
function getNestedValue(obj: Record<string, unknown>, path: string): string {
  return path.split('.').reduce((current: any, key) => {
    return current && current[key] !== undefined ? current[key] : undefined
  }, obj as any) as string
}

/**
 * 国际化函数
 */
export function createI18n(locale: string = 'zh-CN') {
  const currentMessages = messages[locale as keyof typeof messages] || messages['zh-CN']
  
  const t = (key: string, fallback?: string): string => {
    const value = getNestedValue(currentMessages, key)
    return value || fallback || key
  }
  
  return {
    t,
    locale,
    messages: currentMessages
  }
}

/**
 * Vue 3 Composition API Hook
 */
import { computed } from 'vue'
import { useSync } from '../utils/sync'

export function useI18n() {
  const { locale } = useSync()
  
  const i18n = computed(() => createI18n(locale.value))
  
  const t = (key: string, fallback?: string): string => {
    return i18n.value.t(key, fallback)
  }
  
  return {
    t,
    locale,
    messages: computed(() => i18n.value.messages)
  }
}