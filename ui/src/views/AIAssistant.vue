<template>
  <div class="ai-assistant-container">
    <!-- 聊天头部 -->
    <div class="chat-header">
      <div class="header-content">
        <div class="title-section">
          <h1 class="chat-title">
            {{ t('title') }}
          </h1>
          <div class="status-indicator">
            <div 
              class="status-dot"
              :class="{
                'status-connected': aiStore.isConnected,
                'status-error': !aiStore.isConnected
              }"
            ></div>
            <span class="status-text">
              {{ aiStore.isConnected ? t('status.connected') : t('status.error') }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 聊天消息区域 -->
    <div class="chat-messages" ref="messagesContainer">
      <div class="messages-wrapper">
        <!-- 欢迎消息 -->
        <div v-if="messages.length === 0" class="welcome-message">
          <div class="welcome-content">
            <div class="welcome-icon">
              <svg class="w-12 h-12" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v14a2 2 0 002 2z" />
              </svg>
            </div>
            <h3 class="welcome-title">{{ t('welcome.title') }}</h3>
            <p class="welcome-description">{{ t('welcome.description') }}</p>
            <div class="welcome-examples">
              <div class="example-title">{{ t('welcome.examples') }}</div>
              <div class="example-items">
                <button 
                  v-for="example in exampleQueries" 
                  :key="example"
                  class="example-item"
                  @click="sendMessage(example)"
                >
                  {{ example }}
                </button>
              </div>
            </div>
          </div>
        </div>

        <!-- 聊天消息 -->
        <div 
          v-for="(message, index) in messages" 
          :key="index"
          class="message-item"
          :class="{
            'message-user': message.type === 'user',
            'message-assistant': message.type === 'assistant',
            'message-error': message.type === 'error'
          }"
        >
          <div class="message-content">
            <div class="message-avatar">
              <div v-if="message.type === 'user'" class="avatar-user">
                <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path fill-rule="evenodd" d="M10 9a3 3 0 100-6 3 3 0 000 6zm-7 9a7 7 0 1114 0H3z" clip-rule="evenodd" />
                </svg>
              </div>
              <div v-else class="avatar-assistant">
                <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 20 20">
                  <path d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
              </div>
            </div>
            <div class="message-bubble">
              <div class="message-text">{{ message.content }}</div>
              <div v-if="message.sql" class="message-sql">
                <div class="sql-header">
                  <span class="sql-label">{{ t('generated.sql') }}</span>
                  <button class="sql-copy-btn" @click="copySQL(message.sql)">
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
                    </svg>
                  </button>
                </div>
                <pre class="sql-code">{{ message.sql }}</pre>
                <div class="sql-actions">
                  <button class="sql-action-btn" @click="executeSQL(message.sql)">
                    {{ t('actions.execute') }}
                  </button>
                  <button class="sql-action-btn" @click="editSQL(message.sql)">
                    {{ t('actions.edit') }}
                  </button>
                </div>
              </div>
              <div class="message-time">{{ formatTime(message.timestamp) }}</div>
            </div>
          </div>
        </div>

        <!-- 加载指示器 -->
        <div v-if="isLoading" class="message-item message-assistant">
          <div class="message-content">
            <div class="message-avatar">
              <div class="avatar-assistant">
                <div class="loading-spinner"></div>
              </div>
            </div>
            <div class="message-bubble">
              <div class="loading-dots">
                <span></span>
                <span></span>
                <span></span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 输入区域 -->
    <div class="chat-input">
      <div class="input-wrapper">
        <div class="input-container">
          <textarea
            v-model="inputMessage"
            :placeholder="t('input.placeholder')"
            class="message-input"
            rows="1"
            @keydown="handleKeyDown"
            @input="adjustTextareaHeight"
            ref="messageInput"
          ></textarea>
          <button 
            class="send-button"
            :disabled="!inputMessage.trim() || isLoading"
            @click="sendMessage()"
          >
            <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 19l9 2-9-18-9 18 9-2zm0 0v-8" />
            </svg>
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick, watch } from 'vue'
import { useI18n } from '@/i18n'
import { useAIStore } from '@/stores/ai'
// import { useSync } from '@/utils/sync'

// 国际化
const { t } = useI18n()

// 主题同步
// const { language, theme, darkMode } = useSync()

// Stores
const aiStore = useAIStore()

// 响应式数据
const inputMessage = ref('')
const messages = ref<Array<{
  type: 'user' | 'assistant' | 'error'
  content: string
  sql?: string
  timestamp: Date
}>>([])
const isLoading = ref(false)
const messagesContainer = ref<HTMLElement>()
const messageInput = ref<HTMLTextAreaElement>()

// 示例查询
const exampleQueries = computed(() => [
  t('examples.query1'),
  t('examples.query2'),
  t('examples.query3')
])

// 方法
const sendMessage = async (message?: string) => {
  const msg = message || inputMessage.value.trim()
  if (!msg) return

  // 添加用户消息
  messages.value.push({
    type: 'user',
    content: msg,
    timestamp: new Date()
  })

  inputMessage.value = ''
  isLoading.value = true

  try {
      // 调用AI服务生成SQL
      await aiStore.convertToSQL(msg)
      
      // 获取生成的SQL
      const generatedSQL = aiStore.currentSQL
      
      // 添加助手回复
      messages.value.push({
        type: 'assistant',
        content: t('ai.sqlGenerated'),
        sql: generatedSQL,
        timestamp: new Date()
      })
  } catch (error) {
    console.error('AI请求失败:', error)
    messages.value.push({
      type: 'error',
      content: t('errors.aiRequestFailed'),
      timestamp: new Date()
    })
  } finally {
    isLoading.value = false
    await nextTick()
    scrollToBottom()
  }
}

// const handleSQLGenerated = (_sql: string) => {
//   // Handle SQL generation completion
// }

const executeSQL = async (sql: string) => {
  try {
    // 注意：executeSQL方法在当前store中不存在，需要通过API服务直接调用
    // const result = await aiStore.executeSQL(sql)
    // 这里应该调用实际的SQL执行API
    console.log('执行SQL:', sql)
    // 临时实现：显示执行提示
    messages.value.push({
      type: 'assistant',
      content: `SQL执行功能待实现: ${sql}`,
      timestamp: new Date()
    })
  } catch (error) {
    console.error('SQL执行失败:', error)
    messages.value.push({
      type: 'error',
      content: 'SQL执行失败',
      timestamp: new Date()
    })
  }
}

const copySQL = async (sql: string) => {
  try {
    await navigator.clipboard.writeText(sql)
    // 可以添加一个简单的提示
  } catch (error) {
    console.error('复制失败:', error)
  }
}

const editSQL = (sql: string) => {
  inputMessage.value = sql
  messageInput.value?.focus()
}

const scrollToBottom = () => {
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

const handleKeyDown = (event: KeyboardEvent) => {
  if (event.key === 'Enter' && !event.shiftKey) {
    event.preventDefault()
    sendMessage()
  }
}

const adjustTextareaHeight = () => {
  if (messageInput.value) {
    messageInput.value.style.height = 'auto'
    messageInput.value.style.height = messageInput.value.scrollHeight + 'px'
  }
}

const formatTime = (timestamp: Date) => {
  return timestamp.toLocaleTimeString()
}

// 监听输入框变化，自动调整高度
watch(inputMessage, () => {
  nextTick(() => {
    adjustTextareaHeight()
  })
})

// 生命周期
onMounted(async () => {
  try {
    await aiStore.checkHealth()
    await aiStore.loadModelInfo()
    
    // 添加欢迎消息
    messages.value.push({
      type: 'assistant',
      content: t('welcome.message'),
      timestamp: new Date()
    })
  } catch (error) {
    console.error('AI服务初始化失败:', error)
    messages.value.push({
      type: 'error',
      content: t('errors.aiServiceUnavailable'),
      timestamp: new Date()
    })
  }
})
</script>

<style scoped>
.ai-assistant-container {
  @apply h-screen flex flex-col bg-gray-50 dark:bg-gray-900;
}

.chat-header {
  @apply bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700 px-6 py-4;
}

.header-content {
  @apply flex items-center justify-between;
}

.title-section {
  @apply flex items-center space-x-4;
}

.chat-title {
  @apply text-xl font-semibold text-gray-900 dark:text-white;
}

.status-indicator {
  @apply flex items-center space-x-2;
}

.status-dot {
  @apply w-2 h-2 rounded-full;
}

.status-connected {
  @apply bg-green-500;
}

.status-error {
  @apply bg-red-500;
}

.status-text {
  @apply text-sm text-gray-600 dark:text-gray-400;
}

.chat-messages {
  @apply flex-1 overflow-y-auto p-6;
}

.messages-wrapper {
  @apply max-w-4xl mx-auto;
}

.welcome-message {
  @apply text-center py-12;
}

.welcome-content {
  @apply space-y-6;
}

.welcome-icon {
  @apply flex justify-center text-gray-400 dark:text-gray-500;
}

.welcome-title {
  @apply text-2xl font-semibold text-gray-900 dark:text-white;
}

.welcome-description {
  @apply text-gray-600 dark:text-gray-400 max-w-md mx-auto;
}

.welcome-examples {
  @apply space-y-4;
}

.example-title {
  @apply text-sm font-medium text-gray-700 dark:text-gray-300;
}

.example-items {
  @apply space-y-2;
}

.example-item {
  @apply block w-full max-w-md mx-auto px-4 py-3 text-left text-sm bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700 rounded-lg hover:bg-gray-50 dark:hover:bg-gray-700 transition-colors;
}

.message-item {
  @apply mb-6;
}

.message-content {
  @apply flex space-x-3;
}

.message-user .message-content {
  @apply flex-row-reverse space-x-reverse;
}

.message-avatar {
  @apply flex-shrink-0 w-8 h-8 rounded-full flex items-center justify-center;
}

.avatar-user {
  @apply bg-blue-500 text-white;
}

.avatar-assistant {
  @apply bg-gray-500 text-white;
}

.message-bubble {
  @apply max-w-xs lg:max-w-md px-4 py-2 rounded-lg;
}

.message-user .message-bubble {
  @apply bg-blue-500 text-white;
}

.message-assistant .message-bubble {
  @apply bg-white dark:bg-gray-800 border border-gray-200 dark:border-gray-700;
}

.message-error .message-bubble {
  @apply bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 text-red-700 dark:text-red-400;
}

.message-text {
  @apply text-sm;
}

.message-sql {
  @apply mt-3 space-y-2;
}

.sql-header {
  @apply flex items-center justify-between;
}

.sql-label {
  @apply text-xs font-medium text-gray-500 dark:text-gray-400;
}

.sql-copy-btn {
  @apply p-1 text-gray-400 hover:text-gray-600 dark:hover:text-gray-300;
}

.sql-code {
  @apply text-xs bg-gray-100 dark:bg-gray-900 p-3 rounded border font-mono overflow-x-auto;
}

.sql-actions {
  @apply flex space-x-2;
}

.sql-action-btn {
  @apply px-3 py-1 text-xs bg-blue-100 dark:bg-blue-900/30 text-blue-700 dark:text-blue-400 rounded hover:bg-blue-200 dark:hover:bg-blue-900/50;
}

.message-time {
  @apply text-xs text-gray-500 dark:text-gray-400 mt-1;
}

.loading-spinner {
  @apply w-4 h-4 border-2 border-gray-300 border-t-gray-600 rounded-full animate-spin;
}

.loading-dots {
  @apply flex space-x-1;
}

.loading-dots span {
  @apply w-2 h-2 bg-gray-400 rounded-full animate-pulse;
}

.loading-dots span:nth-child(2) {
  @apply animation-delay-75;
}

.loading-dots span:nth-child(3) {
  @apply animation-delay-150;
}

.chat-input {
  @apply border-t border-gray-200 dark:border-gray-700 bg-white dark:bg-gray-800 p-4;
}

.input-wrapper {
  @apply max-w-4xl mx-auto;
}

.input-container {
  @apply flex space-x-3;
}

.message-input {
  @apply flex-1 resize-none border border-gray-300 dark:border-gray-600 rounded-lg px-4 py-2 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-transparent bg-white dark:bg-gray-700 text-gray-900 dark:text-white;
}

.send-button {
  @apply px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 disabled:opacity-50 disabled:cursor-not-allowed flex items-center justify-center;
}

/* 响应式设计 */
@media (max-width: 640px) {
  .chat-header {
    @apply px-4 py-3;
  }
  
  .chat-messages {
    @apply p-4;
  }
  
  .message-bubble {
    @apply max-w-xs;
  }
  
  .welcome-message {
    @apply py-8;
  }
}
</style>