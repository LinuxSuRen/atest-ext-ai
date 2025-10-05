<template>
  <div class="chat-messages" ref="messagesContainer">
    <div
      v-for="message in props.messages"
      :key="message.id"
      :class="['message', `message-${message.type}`]"
    >
      <div class="message-content">
        <div class="message-text">{{ message.content }}</div>
        <div v-if="message.sql" class="message-sql">
          <pre>{{ message.sql }}</pre>
          <el-button
            size="small"
            type="primary"
            @click="copySQL(message.sql)"
          >
            <el-icon><DocumentCopy /></el-icon>
            {{ t('button.copy') }}
          </el-button>
        </div>
        <div v-if="message.meta" class="message-meta">
          <el-tag size="small">{{ message.meta.model }}</el-tag>
          <span class="meta-time">{{ message.meta.duration }}ms</span>
        </div>
      </div>
      <div class="message-time">
        {{ formatTime(message.timestamp) }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, inject, watch, nextTick } from 'vue'
import { DocumentCopy } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { AppContext, Message } from '../types'

interface Props {
  messages: Message[]
}
const props = defineProps<Props>()

// Inject context
const context = inject<AppContext>('appContext')!
const { t } = context.i18n

// Message container ref for auto-scroll
const messagesContainer = ref<HTMLElement>()

// Watch messages and scroll to bottom
watch(() => props.messages.length, async () => {
  await nextTick()
  scrollToBottom()
})

function scrollToBottom() {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

function formatTime(timestamp: number): string {
  const date = new Date(timestamp)
  return date.toLocaleTimeString()
}

async function copySQL(sql: string) {
  try {
    await navigator.clipboard.writeText(sql)
    ElMessage.success(t('ai.message.copiedSuccess'))
  } catch (error) {
    ElMessage.error('Failed to copy')
  }
}
</script>

<style scoped>
.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 16px 24px;
}

.message {
  margin-bottom: 16px;
  padding: 12px;
  border-radius: 8px;
  max-width: 80%;
}

.message-user {
  margin-left: auto;
  background: #409eff;
  color: #fff;
}

.message-ai {
  background: #fff;
  border: 1px solid #e4e7ed;
}

.message-error {
  background: #fef0f0;
  border: 1px solid #fbc4c4;
  color: #f56c6c;
}

.message-text {
  margin-bottom: 8px;
  line-height: 1.6;
}

.message-sql {
  margin-top: 12px;
  padding: 12px;
  background: #f5f7fa;
  border-radius: 4px;
  position: relative;
}

.message-sql pre {
  margin: 0;
  font-family: 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  white-space: pre-wrap;
  word-wrap: break-word;
}

.message-sql .el-button {
  margin-top: 8px;
}

.message-meta {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 8px;
  font-size: 12px;
  color: #909399;
}

.message-time {
  margin-top: 4px;
  font-size: 12px;
  color: #c0c4cc;
  text-align: right;
}
</style>
