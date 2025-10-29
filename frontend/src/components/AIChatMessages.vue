<template>
  <div class="chat-messages" ref="messagesContainer">
    <!-- Empty state -->
    <div v-if="props.messages.length === 0" class="empty-state">
      <el-icon :size="64" color="var(--el-text-color-placeholder)">
        <ChatDotRound />
      </el-icon>
      <p>{{ t('ai.welcome.startChat') }}</p>
    </div>

    <!-- Messages list -->
    <div
      v-for="message in props.messages"
      :key="message.id"
      :class="['message-wrapper', `message-${message.type}`]"
    >
      <!-- AI Message: Avatar on left -->
      <div v-if="message.type === 'ai'" class="message-avatar">
        <el-avatar :size="36" class="avatar-ai">
          <el-icon :size="20"><ChatDotRound /></el-icon>
        </el-avatar>
      </div>

      <!-- Message bubble -->
      <div class="message-bubble">
        <div class="message-content">
          <div class="message-text">{{ message.content }}</div>
          <div v-if="message.sql" class="message-sql">
            <div class="sql-header">
              <span class="sql-label">SQL</span>
              <el-button
                link
                size="small"
                @click="copySQL(message.sql)"
                class="copy-btn"
              >
                <el-icon><DocumentCopy /></el-icon>
                {{ t('ai.button.copy') }}
              </el-button>
            </div>
            <pre class="sql-code">{{ message.sql }}</pre>
          </div>
          <div v-if="message.meta" class="message-meta">
            <el-tag size="small" effect="plain">{{ message.meta.model }}</el-tag>
            <span class="meta-time">{{ message.meta.duration }}ms</span>
          </div>
        </div>
        <div class="message-time">
          {{ formatTime(message.timestamp) }}
        </div>
      </div>

      <!-- User Message: Avatar on right -->
      <div v-if="message.type === 'user'" class="message-avatar">
        <el-avatar :size="36" class="avatar-user">
          <el-icon :size="20"><User /></el-icon>
        </el-avatar>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, inject, watch, nextTick } from 'vue'
import { DocumentCopy, ChatDotRound, User } from '@element-plus/icons-vue'
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
  height: 100%;
  min-height: 0;
  overflow-y: auto;
  padding: 24px 40px;
  background: var(--atest-bg-surface);
  color: var(--atest-text-primary);
  border: 1px solid var(--atest-border-color);
  border-radius: var(--atest-radius-md);
}

/* Empty state */
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  min-height: 300px;
  gap: 16px;
}

.empty-state p {
  color: var(--atest-text-secondary);
  font-size: 14px;
  margin: 0;
}

/* Message wrapper with avatar */
.message-wrapper {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
  align-items: flex-start;
}

.message-wrapper.message-user {
  flex-direction: row-reverse;
}

/* Avatar */
.message-avatar {
  flex-shrink: 0;
}

.avatar-ai {
  background: var(--el-color-primary);
  color: #fff;
}

.avatar-user {
  background: var(--el-color-success);
  color: #fff;
}

/* Message bubble */
.message-bubble {
  flex: 1;
  max-width: calc(100% - 60px);
  min-width: 200px;
}

.message-ai .message-bubble {
  margin-right: auto;
}

.message-user .message-bubble {
  margin-left: auto;
}

/* Bubble content */
.message-content {
  padding: 14px 18px;
  border-radius: var(--atest-radius-lg);
  position: relative;
  box-shadow: var(--atest-shadow-sm);
  animation: fadeIn 0.3s ease-in-out;
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(10px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

/* AI message bubble (white) */
.message-ai .message-content {
  background: var(--atest-bg-surface);
  border-bottom-left-radius: 4px;
}

@media (max-width: 1024px) {
  .chat-messages {
    padding: 20px 24px;
  }
}

@media (max-width: 768px) {
  .chat-messages {
    padding: 16px 18px;
  }

  .message-wrapper {
    gap: 10px;
  }

  .message-bubble {
    max-width: 100%;
    min-width: 0;
  }

  .message-content {
    padding: 12px 14px;
  }
}

@media (max-width: 480px) {
  .chat-messages {
    padding: 12px 12px;
  }

  .message-content {
    padding: 10px 12px;
  }

  .message-wrapper {
    margin-bottom: 16px;
  }
}

/* User message bubble */
.message-user .message-content {
  background: var(--atest-color-accent);
  color: #fff;
  border-bottom-right-radius: 4px;
}

/* Error message bubble */
.message-error .message-content {
  background: var(--atest-color-danger-soft);
  border: 1px solid var(--el-color-danger-light-7, rgba(245, 108, 108, 0.3));
  color: var(--el-color-danger, #f56c6c);
}

/* Message text */
.message-text {
  line-height: 1.6;
  font-size: 14px;
  word-wrap: break-word;
}

/* SQL code block */
.message-sql {
  margin-top: 12px;
  background: color-mix(in srgb, var(--atest-color-accent) 12%, #1f2933);
  border-radius: 8px;
  overflow: hidden;
}

.sql-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 12px;
  background: color-mix(in srgb, var(--atest-color-accent) 18%, #111827);
  border-bottom: 1px solid transparent;
}

.sql-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--atest-text-regular);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.copy-btn {
  color: var(--atest-color-accent) !important;
  font-size: 12px;
}

.copy-btn:hover {
  color: var(--el-color-primary-light-3) !important;
}

.sql-code {
  margin: 0;
  padding: 12px;
  font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.5;
  color: var(--atest-text-primary);
  white-space: pre-wrap;
  word-wrap: break-word;
  background: color-mix(in srgb, var(--atest-bg-surface) 85%, #000 15%);
}

/* Message metadata */
.message-meta {
  display: flex;
  gap: 8px;
  align-items: center;
  margin-top: 8px;
  font-size: 12px;
}

.message-ai .message-meta {
  color: var(--atest-text-regular);
}

.message-user .message-meta {
  color: rgba(255, 255, 255, 0.9);
}

.message-user .message-meta :deep(.el-tag) {
  background: rgba(255, 255, 255, 0.2);
  border-color: rgba(255, 255, 255, 0.3);
  color: #fff;
}

.meta-time {
  font-size: 11px;
}

/* Message timestamp */
.message-time {
  margin-top: 6px;
  font-size: 11px;
  color: var(--el-text-color-placeholder);
  padding: 0 4px;
}

.message-user .message-time {
  text-align: right;
}

.message-ai .message-time {
  text-align: left;
}

/* Scrollbar styling */
.chat-messages::-webkit-scrollbar {
  width: 8px;
}

.chat-messages::-webkit-scrollbar-track {
  background: transparent;
}

.chat-messages::-webkit-scrollbar-thumb {
  background: var(--el-border-color);
  border-radius: 4px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: var(--el-border-color-darker);
}
</style>
