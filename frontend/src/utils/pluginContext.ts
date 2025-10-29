import { ref, computed, type Ref } from 'vue'
import type { AppContext } from '@/types'
import { createTranslator, normalizeLocale } from './i18n'

function detectBrowserLocale(): string {
  if (typeof navigator === 'undefined' || !navigator.language) {
    return 'en'
  }
  return normalizeLocale(navigator.language)
}

export interface PluginContextBridge {
  context: AppContext
  setLocale: (locale: string) => void
  locale: Ref<string>
}

export function createPluginContextBridge(provided?: AppContext): PluginContextBridge {
  const fallbackLocale = ref(detectBrowserLocale())
  const localeRef = provided?.i18n?.locale ?? fallbackLocale
  const baseI18n = provided?.i18n ?? {
    t: (key: string) => key,
    locale: localeRef
  }

  const translator = computed(() => {
    // ensure dependency tracking on locale changes
    // eslint-disable-next-line @typescript-eslint/no-unused-expressions
    baseI18n.locale.value
    return createTranslator(baseI18n)
  })

  const context: AppContext = {
    i18n: {
      locale: localeRef,
      t: (key: string) => translator.value(key)
    },
    API: provided?.API ?? {},
    Cache: provided?.Cache ?? {}
  }

  const setLocale = (locale: string) => {
    localeRef.value = normalizeLocale(locale)
  }

  return {
    context,
    setLocale,
    locale: localeRef
  }
}
