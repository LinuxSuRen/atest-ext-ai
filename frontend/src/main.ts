import { createApp, type App as VueApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import type { AppContext } from './types'
import './styles/tokens.css'
import { createPluginContextBridge, type PluginContextBridge } from './utils/pluginContext'
import { normalizeLocale } from './utils/i18n'

// Store Vue app instance for potential cleanup
let app: VueApp | null = null
let bridge: PluginContextBridge | null = null
let pendingLocale: string | null = null

/**
 * Plugin interface exposed to main application
 */
const ATestPlugin = {
  /**
   * Mount plugin with context from main app
   * @param el - DOM container element or selector string
   * @param context - Optional context from main app (i18n, API, Cache)
   */
  mount(el?: string | Element, context?: AppContext) {
    const container = typeof el === 'string' ? document.querySelector(el) : el;
    // Cleanup previous instance if exists
    if (app) {
      app.unmount()
    }

    bridge = createPluginContextBridge(context)

    // Create new Vue app with context passed as props
    app = createApp(App, { context: bridge.context })

    // Use Element Plus
    app.use(ElementPlus)

    // Mount to container
    app.mount(container)

    if (pendingLocale) {
      bridge.setLocale(pendingLocale)
      pendingLocale = null
    }
  },

  /**
   * Unmount plugin (for cleanup)
   */
  unmount() {
    if (app) {
      app.unmount()
      app = null
    }
    bridge = null
  },

  /**
   * Allow host application to toggle locale proactively
   */
  setLocale(locale: string) {
    const normalized = normalizeLocale(locale)
    if (bridge) {
      bridge.setLocale(normalized)
    } else {
      pendingLocale = normalized
    }
  }
}

// Expose plugin to window for main app to access
declare global {
  interface Window {
    ATestPlugin: typeof ATestPlugin
  }
}

if (sessionStorage.getItem('mode') === 'dev') {
  ATestPlugin.mount('#plugin-container');
}

window.ATestPlugin = ATestPlugin

export default ATestPlugin
