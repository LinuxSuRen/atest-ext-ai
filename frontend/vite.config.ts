import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],

  // Define global constants for browser environment
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },

  build: {
    lib: {
      entry: resolve(__dirname, 'src/main.ts'),
      name: 'ATestPlugin',
      formats: ['umd'],
      // Output filename must match what Go embed expects
      fileName: () => 'ai-chat.js'
    },
    // Output directly to assets directory for Go embed
    // The backend uses //go:embed to bundle these files into the binary
    outDir: '../pkg/plugin/assets',
    emptyOutDir: false,

    rollupOptions: {
      // No external dependencies - bundle everything
      external: [],
      output: {
        // UMD format will automatically use correct global object (window)
        globals: {},
        // CSS filename must match what Go embed expects
        assetFileNames: 'ai-chat.css',
        // Inject process polyfill at the start of the UMD bundle
        intro: 'var process = { env: { NODE_ENV: "production" } };'
      }
    }
  },

  resolve: {
    alias: {
      '@': resolve(__dirname, './src')
    }
  }
})
