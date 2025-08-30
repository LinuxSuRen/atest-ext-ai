import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

export interface AppSettings {
  theme: 'light' | 'dark' | 'auto'
  language: 'zh-CN' | 'en-US'
  autoSave: boolean
  showExplanation: boolean
  maxHistoryItems: number
  apiTimeout: number
}

const DEFAULT_SETTINGS: AppSettings = {
  theme: 'auto',
  language: 'zh-CN',
  autoSave: true,
  showExplanation: true,
  maxHistoryItems: 50,
  apiTimeout: 30000,
}

const STORAGE_KEY = 'ai-plugin-settings'

export const useSettingsStore = defineStore('settings', () => {
  // 状态
  const settings = ref<AppSettings>({ ...DEFAULT_SETTINGS })
  const isLoading = ref(false)

  // 从本地存储加载设置
  const loadSettings = () => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY)
      if (stored) {
        const parsedSettings = JSON.parse(stored)
        settings.value = { ...DEFAULT_SETTINGS, ...parsedSettings }
      }
    } catch (error) {
      console.error('Failed to load settings:', error)
      settings.value = { ...DEFAULT_SETTINGS }
    }
  }

  // 保存设置到本地存储
  const saveSettings = () => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(settings.value))
    } catch (error) {
      console.error('Failed to save settings:', error)
    }
  }

  // 更新设置
  const updateSettings = (newSettings: Partial<AppSettings>) => {
    settings.value = { ...settings.value, ...newSettings }
    saveSettings()
  }

  // 重置设置
  const resetSettings = () => {
    settings.value = { ...DEFAULT_SETTINGS }
    saveSettings()
  }

  // 获取特定设置
  const getSetting = <K extends keyof AppSettings>(key: K): AppSettings[K] => {
    return settings.value[key]
  }

  // 设置特定值
  const setSetting = <K extends keyof AppSettings>(key: K, value: AppSettings[K]) => {
    settings.value[key] = value
    saveSettings()
  }

  // 计算属性 - 直接访问设置值
  const theme = computed(() => settings.value.theme)
  const language = computed(() => settings.value.language)
  const autoSave = computed(() => settings.value.autoSave)
  const showExplanation = computed(() => settings.value.showExplanation)
  const maxHistoryItems = computed(() => settings.value.maxHistoryItems)
  const historyLimit = computed(() => settings.value.maxHistoryItems)
  const apiTimeout = computed(() => settings.value.apiTimeout)

  // 添加updateSetting方法（别名）
  const updateSetting = setSetting

  // 初始化时加载设置
  loadSettings()

  return {
    // 状态
    settings,
    isLoading,
    
    // 计算属性
    theme,
    language,
    autoSave,
    showExplanation,
    maxHistoryItems,
    historyLimit,
    apiTimeout,
    
    // 方法
    loadSettings,
    saveSettings,
    updateSettings,
    updateSetting,
    resetSettings,
    getSetting,
    setSetting,
  }
})