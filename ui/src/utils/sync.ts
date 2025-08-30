import { ref, watch, onUnmounted } from 'vue'
import { getMainProjectPreference, watchMainProjectSettings, applyDarkMode } from './themeSync'

/**
 * 与主项目同步的工具函数
 */

// 同步状态
const syncState = ref({
  language: 'zh-CN',
  theme: 'light',
  darkMode: false
})

// 初始化同步状态
function initSyncState() {
  const preference = getMainProjectPreference()
  syncState.value = {
    language: preference.language || 'zh-CN',
    theme: preference.darkTheme ? 'dark' : 'light',
    darkMode: preference.darkTheme
  }
  
  // 应用暗黑模式
  applyDarkMode(preference.darkTheme)
}

// 初始化
initSyncState()

/**
 * 获取当前语言设置
 */
export function getCurrentLanguage(): string {
  return syncState.value.language
}

/**
 * 获取当前主题设置
 */
export function getCurrentTheme(): string {
  return syncState.value.theme
}

/**
 * 获取当前暗黑模式状态
 */
export function getCurrentDarkMode(): boolean {
  return syncState.value.darkMode
}

/**
 * 监听语言变化
 */
export function watchLanguageChange(callback: (language: string) => void) {
  return watch(() => syncState.value.language, callback, { immediate: true })
}

/**
 * 监听主题变化
 */
export function watchThemeChange(callback: (theme: string) => void) {
  return watch(() => syncState.value.theme, callback, { immediate: true })
}

/**
 * 监听暗黑模式变化
 */
export function watchDarkModeChange(callback: (darkMode: boolean) => void) {
  return watch(() => syncState.value.darkMode, callback, { immediate: true })
}

/**
 * 同步工具的 Composition API Hook
 */
export function useSync() {
  const language = ref(getCurrentLanguage())
  const theme = ref(getCurrentTheme())
  const darkMode = ref(getCurrentDarkMode())

  // 监听主项目设置变化
  const cleanup = watchMainProjectSettings((preference) => {
    syncState.value = {
      language: preference.language || 'zh-CN',
      theme: preference.darkTheme ? 'dark' : 'light',
      darkMode: preference.darkTheme
    }
    
    language.value = syncState.value.language
    theme.value = syncState.value.theme
    darkMode.value = syncState.value.darkMode
    
    // 应用暗黑模式
    applyDarkMode(preference.darkTheme)
  })

  // 清理函数
  onUnmounted(() => {
    cleanup()
  })

  return {
    language,
    theme,
    darkMode,
    locale: language, // 添加locale别名，与language保持一致
    isDark: darkMode, // 添加isDark别名，与darkMode保持一致
    config: { theme, language } // 添加config对象
  }
}