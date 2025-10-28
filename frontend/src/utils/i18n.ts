import type { AppContext } from '@/types'
import en from '@/locales/en.json'
import zh from '@/locales/zh.json'

const messages: Record<string, any> = {
  en,
  zh
}

function normalizeLocale(locale: string | undefined): string {
  if (!locale) {
    return 'en'
  }

  const lower = locale.toLowerCase()

  if (lower.startsWith('zh')) {
    return 'zh'
  }

  if (lower.startsWith('en')) {
    return 'en'
  }

  const [base] = lower.split('-')
  return base || 'en'
}

function resolveMessage(locale: string, key: string): string | undefined {
  const segments = key.split('.')
  const localeMessages = messages[locale] || messages.en
  let current: any = localeMessages

  for (const segment of segments) {
    if (current && typeof current === 'object' && segment in current) {
      current = current[segment]
    } else {
      return undefined
    }
  }

  return typeof current === 'string' ? current : undefined
}

/**
 * Create translator that falls back to plugin-local messages when host app
 * does not provide a translation key.
 */
export function createTranslator(i18n: AppContext['i18n']) {
  return (key: string): string => {
    const hostValue = i18n.t(key)
    if (hostValue !== key) {
      return hostValue
    }

    const locale = normalizeLocale(i18n.locale.value)
    return resolveMessage(locale, key) ?? resolveMessage('en', key) ?? key
  }
}
