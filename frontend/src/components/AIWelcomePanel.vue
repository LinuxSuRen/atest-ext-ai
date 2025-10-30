<template>
  <div class="welcome-panel">
    <el-empty :description="t('ai.welcome.message')">
      <template #image>
        <el-icon :size="80"><Setting /></el-icon>
      </template>
      <el-button type="primary" @click="emit('configure')">
        {{ t('ai.button.configure') }}
      </el-button>
    </el-empty>
  </div>
</template>

<script setup lang="ts">
import { inject } from 'vue'
import { Setting } from '@element-plus/icons-vue'
import type { AppContext } from '../types'

interface Emits {
  (e: 'configure'): void
}
const emit = defineEmits<Emits>()

// Inject context
const context = inject<AppContext>('appContext')!
const { t } = context.i18n
</script>

<style scoped>
.welcome-panel {
  padding: 48px;
  text-align: center;
}
</style>
