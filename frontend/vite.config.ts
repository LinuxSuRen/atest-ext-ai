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
      fileName: () => 'main.js'
    },
    outDir: '../dist',
    emptyOutDir: false,

    rollupOptions: {
      // No external dependencies - bundle everything
      external: [],
      output: {
        // Expose plugin on window
        extend: true,
        globals: {}
      }
    }
  },

  resolve: {
    alias: {
      '@': resolve(__dirname, './src')
    }
  }
})
