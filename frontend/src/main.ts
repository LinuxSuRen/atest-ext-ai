import { createApp, App as VueApp } from 'vue'
import ElementPlus from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import type { AppContext } from './types'

// Store Vue app instance for potential cleanup
let app: VueApp | null = null

/**
 * Plugin interface exposed to main application
 */
const ATestPlugin = {
  /**
   * Mount plugin with context from main app
   * @param container - DOM container element
   * @param context - Context from main app (i18n, API, Cache)
   */
  mount(container: HTMLElement, context: AppContext) {
    // Cleanup previous instance if exists
    if (app) {
      app.unmount()
    }

    // Create new Vue app with context passed as props
    app = createApp(App, { context })

    // Use Element Plus
    app.use(ElementPlus)

    // Mount to container
    app.mount(container)
  },

  /**
   * Unmount plugin (for cleanup)
   */
  unmount() {
    if (app) {
      app.unmount()
      app = null
    }
  }
}

// Expose plugin to window for main app to access
declare global {
  interface Window {
    ATestPlugin: typeof ATestPlugin
  }
}

window.ATestPlugin = ATestPlugin

export default ATestPlugin
