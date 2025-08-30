/**
 * 主题同步工具
 * 用于与主项目的主题设置保持同步
 */

interface Preference {
  darkTheme: boolean
  language: string
  requestActiveTab: string
  responseActiveTab: string
}

const PREFERENCE_KEY = 'api-testing-preference'
const THEME_KEY = 'theme'

/**
 * 获取主项目的偏好设置
 */
export function getMainProjectPreference(): Preference {
  const val = localStorage.getItem(PREFERENCE_KEY)
  if (val && val !== '') {
    return JSON.parse(val)
  } else {
    const navLanguage = navigator.language != null ? navigator.language : 'zh-CN'
    return {
      darkTheme: false,
      language: navLanguage,
      requestActiveTab: 'body',
      responseActiveTab: 'body'
    }
  }
}

/**
 * 获取主项目的主题设置
 */
export function getMainProjectTheme(): string | null {
  return localStorage.getItem(THEME_KEY)
}

/**
 * 应用暗黑模式
 */
export function applyDarkMode(darkMode: boolean) {
  document.documentElement.className = darkMode ? 'dark' : 'light'
  
  // 同时设置data-theme属性，兼容不同的主题系统
  document.documentElement.setAttribute('data-theme', darkMode ? 'dark' : 'light')
}

/**
 * 监听主项目设置变化
 */
export function watchMainProjectSettings(callback: (preference: Preference, theme: string | null) => void) {
  // 监听localStorage变化
  const handleStorageChange = (e: StorageEvent) => {
    if (e.key === PREFERENCE_KEY || e.key === THEME_KEY) {
      const preference = getMainProjectPreference()
      const theme = getMainProjectTheme()
      callback(preference, theme)
    }
  }

  window.addEventListener('storage', handleStorageChange)

  // 返回清理函数
  return () => {
    window.removeEventListener('storage', handleStorageChange)
  }
}

/**
 * 初始化主题同步
 */
export function initThemeSync() {
  const preference = getMainProjectPreference()
  const theme = getMainProjectTheme()
  
  // 应用暗黑模式
  applyDarkMode(preference.darkTheme)
  
  // 如果有自定义主题，应用主题
  if (theme) {
    applyCustomTheme(theme)
  }
  
  return { preference, theme }
}

/**
 * 应用自定义主题（如果需要）
 */
function applyCustomTheme(themeName: string) {
  // 这里可以根据需要实现自定义主题的应用逻辑
  // 目前主要依赖CSS变量和暗黑模式类名
  console.log('Applied custom theme:', themeName)
}

/**
 * 创建主题同步的Composition API Hook
 */
export function useThemeSync() {
  const { preference, theme } = initThemeSync()
  
  // 监听设置变化
  const cleanup = watchMainProjectSettings((newPreference, newTheme) => {
    applyDarkMode(newPreference.darkTheme)
    if (newTheme) {
      applyCustomTheme(newTheme)
    }
  })
  
  return {
    preference,
    theme,
    cleanup
  }
}