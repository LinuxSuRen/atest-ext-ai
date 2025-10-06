import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [vue()],

  build: {
    lib: {
      entry: resolve(__dirname, 'src/main.ts'),
      name: 'ATestPlugin',
      formats: ['iife'],
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
        // Expose plugin on window
        extend: true,
        globals: {},
        // CSS filename must match what Go embed expects
        assetFileNames: 'ai-chat.css'
      }
    }
  },

  resolve: {
    alias: {
      '@': resolve(__dirname, './src')
    }
  }
})
