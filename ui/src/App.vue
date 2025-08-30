<script setup lang="ts">
import { RouterView } from 'vue-router'
import { useSync } from './utils/sync'
import { onMounted } from 'vue'

// 使用同步功能
// const { language, theme } = useSync()
const { darkMode } = useSync()

// 初始化时应用主题
onMounted(() => {
  // 确保主题正确应用
  document.documentElement.className = darkMode.value ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', darkMode.value ? 'dark' : 'light')
})
</script>

<template>
  <div id="app">
    <RouterView />
  </div>
</template>

<style>
#app {
  height: 100vh;
  width: 100vw;
  margin: 0;
  padding: 0;
}

* {
  box-sizing: border-box;
}

body {
  margin: 0;
  padding: 0;
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'Roboto', 'Oxygen',
    'Ubuntu', 'Cantarell', 'Fira Sans', 'Droid Sans', 'Helvetica Neue',
    sans-serif;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>
