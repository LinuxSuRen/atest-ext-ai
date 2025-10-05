<template>
  <div class="chat-input">
    <div class="input-options">
      <el-checkbox v-model="includeExplanation">
        {{ t('ai.option.includeExplanation') }}
      </el-checkbox>
    </div>
    <div class="input-controls">
      <el-input
        v-model="prompt"
        type="textarea"
        :rows="3"
        :placeholder="t('ai.input.placeholder')"
        :disabled="props.loading"
        @keydown.enter.ctrl="handleSubmit"
        @keydown.enter.meta="handleSubmit"
      />
      <el-button
        type="primary"
        :loading="props.loading"
        :disabled="!prompt.trim()"
        @click="handleSubmit"
      >
        <el-icon v-if="!props.loading"><Promotion /></el-icon>
        {{ props.loading ? t('ai.message.generating') : t('ai.button.generate') }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, inject } from 'vue'
import { Promotion } from '@element-plus/icons-vue'
import type { AppContext } from '../types'

interface Props {
  loading: boolean
}
const props = defineProps<Props>()

interface Emits {
  (e: 'submit', prompt: string, options: { includeExplanation: boolean }): void
}
const emit = defineEmits<Emits>()

// Inject context
const context = inject<AppContext>('appContext')!
const { t } = context.i18n

// Input state
const prompt = ref('')
const includeExplanation = ref(false)

function handleSubmit() {
  if (!prompt.value.trim() || props.loading) return

  emit('submit', prompt.value, {
    includeExplanation: includeExplanation.value
  })

  // Clear input after submit
  prompt.value = ''
}
</script>

<style scoped>
.chat-input {
  padding: 16px 24px;
  background: #fff;
  border-top: 1px solid #e4e7ed;
}

.input-options {
  margin-bottom: 12px;
}

.input-controls {
  display: flex;
  gap: 12px;
  align-items: flex-end;
}

.input-controls .el-input {
  flex: 1;
}

.input-controls .el-button {
  height: 40px;
  white-space: nowrap;
}
</style>
