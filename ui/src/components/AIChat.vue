<template>
  <div class="ai-chat-container">
    <!-- 聊天头部 -->
    <div class="chat-header">
      <div class="flex items-center justify-between">
        <div class="flex items-center space-x-3">
          <div class="w-8 h-8 bg-blue-500 rounded-full flex items-center justify-center">
            <span class="text-white text-sm font-bold">AI</span>
          </div>
          <div>
            <h2 class="text-lg font-semibold text-gray-800 dark:text-gray-200">
              {{ t('ai.title') }}
            </h2>
            <div class="flex items-center space-x-2">
              <div 
                :class="[
                  'w-2 h-2 rounded-full',
                  isInFallbackMode ? 'bg-red-500' : 'bg-green-500'
                ]"
              ></div>
              <span :class="isInFallbackMode ? 'text-red-500' : 'text-green-500'">
                {{ isInFallbackMode ? t('ai.status.error') : t('ai.status.connected') }}
              </span>
            </div>
          </div>
        </div>
        <el-button 
          v-if="chatMessages.length > 0"
          type="text" 
          size="small"
          @click="clearChat"
          class="text-gray-500 hover:text-gray-700"
        >
          {{ t('ai.clear') }}
        </el-button>
      </div>
    </div>

    <!-- 降级模式提示 -->
    <div 
      v-if="isInFallbackMode" 
      class="mx-4 mt-4 p-3 bg-yellow-50 dark:bg-yellow-900/20 border border-yellow-200 dark:border-yellow-800 rounded-lg"
    >
      <div class="flex items-start space-x-2">
        <el-icon class="text-yellow-600 dark:text-yellow-400 mt-0.5">
          <WarningFilled />
        </el-icon>
        <div class="flex-1">
          <h4 class="text-sm font-medium text-yellow-800 dark:text-yellow-200">
            {{ t('ai.fallback.title') }}
          </h4>
          <p class="text-xs text-yellow-700 dark:text-yellow-300 mt-1">
            {{ t('ai.fallback.description') }}
          </p>
          <button 
            v-if="fallbackStatus.canRetry"
            @click="resetFallbackState"
            class="mt-2 text-xs text-yellow-800 dark:text-yellow-200 underline hover:no-underline"
          >
            重新尝试连接
          </button>
        </div>
      </div>
    </div>

    <!-- 聊天消息区域 -->
    <div class="chat-messages" ref="messagesContainer">
      <!-- 欢迎消息 -->
      <div v-if="chatMessages.length === 0" class="welcome-message">
        <div class="message-bubble ai-message">
          <div class="message-content">
            <p class="text-gray-700 dark:text-gray-300 mb-3">
              👋 您好！我是AI SQL助手，可以帮您将自然语言转换为SQL查询。
            </p>
            <p class="text-gray-600 dark:text-gray-400 text-sm">
              请描述您想要查询的内容，例如：
            </p>
            <div class="mt-3 space-y-2">
              <div 
                v-for="example in quickExamples" 
                :key="example.id"
                class="example-item"
                @click="useExample(example.query)"
              >
                <span class="text-blue-600 dark:text-blue-400 text-sm cursor-pointer hover:underline">
                  {{ example.query }}
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 聊天消息列表 -->
      <div v-for="message in chatMessages" :key="message.id" class="message-wrapper">
        <!-- 用户消息 -->
        <div v-if="message.type === 'user'" class="message-bubble user-message">
          <div class="message-content">
            <div class="font-medium text-blue-600 dark:text-blue-400 mb-1">
              {{ t('ai.message.user') }}
            </div>
            <p class="text-white">{{ message.content }}</p>
            <div class="message-time">
              {{ formatTime(message.timestamp) }}
            </div>
          </div>
        </div>

        <!-- AI消息 -->
        <div v-else class="message-bubble ai-message">
          <div class="message-content">
            <div class="font-medium text-green-600 dark:text-green-400 mb-2">
              {{ t('ai.message.assistant') }}
            </div>
            <!-- 加载状态 -->
            <div v-if="message.loading" class="flex items-center space-x-2">
              <div class="loading-dots">
                <div class="dot"></div>
                <div class="dot"></div>
                <div class="dot"></div>
              </div>
              <span class="text-gray-500 text-sm">正在生成SQL...</span>
            </div>
            
            <!-- 错误消息 -->
            <div v-else-if="message.error" class="error-content">
              <div class="flex items-center space-x-2 text-red-600 mb-2">
                <el-icon><WarningFilled /></el-icon>
                <span class="font-medium">转换失败</span>
              </div>
              <p class="text-red-700 dark:text-red-400 text-sm">{{ message.error }}</p>
            </div>
            
            <!-- 成功消息 -->
            <div v-else class="success-content">
              <div class="mb-3">
                <p class="text-gray-700 dark:text-gray-300 mb-2">已为您生成SQL查询：</p>
              </div>
              
              <!-- SQL代码块 -->
              <div class="sql-block">
                <div class="flex items-center justify-between mb-2">
                  <span class="text-xs font-medium text-gray-600 dark:text-gray-400">SQL</span>
                  <div class="flex space-x-2">
                    <el-button 
                      type="text" 
                      size="small"
                      @click="copySQL(message.sql!)"
                      class="text-xs"
                    >
                      {{ t('ai.copy') }}
                    </el-button>
                    <el-button 
                      type="text" 
                      size="small"
                      @click="executeSQL(message.sql!)"
                      class="text-xs"
                    >
                      {{ t('ai.execute') }}
                    </el-button>
                  </div>
                </div>
                <pre class="sql-code">{{ message.sql }}</pre>
              </div>
              
              <!-- 解释说明 -->
              <div v-if="message.explanation" class="explanation-block">
                <div class="text-xs font-medium text-gray-600 dark:text-gray-400 mb-2">解释</div>
                <p class="text-gray-600 dark:text-gray-400 text-sm">{{ message.explanation }}</p>
              </div>
            </div>
            
            <div class="message-time">
              {{ formatTime(message.timestamp) }}
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 输入区域 -->
    <div class="chat-input">
      <div class="input-container">
        <el-input
          v-model="queryInput"
          type="textarea"
          :rows="1"
          :autosize="{ minRows: 1, maxRows: 4 }"
          :placeholder="t('ai.placeholder')"
          :disabled="aiStore.isLoading"
          @keydown.enter.exact.prevent="handleSubmit"
          @keydown.shift.enter.exact="handleNewLine"
          class="message-input"
        />
        <el-button 
          type="primary"
          :loading="aiStore.isLoading"
          :disabled="!queryInput.trim()"
          @click="handleSubmit"
          class="send-button"
        >
          <el-icon v-if="!aiStore.isLoading"><Promotion /></el-icon>
          <span v-if="aiStore.isLoading">{{ t('ai.loading') }}</span>
        </el-button>
      </div>
      <div class="input-hint">
        <span class="text-xs text-gray-500">按 Enter 发送，Shift + Enter 换行</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, nextTick } from 'vue'
import { useAIStore } from '../stores/ai'
// import { useSettingsStore } from '../stores/settings'
import { ElMessage } from 'element-plus'
import { WarningFilled, Promotion } from '@element-plus/icons-vue'
// import { useSync } from '../utils/sync'
import { useI18n } from '../i18n'

interface ChatMessage {
  id: string
  type: 'user' | 'ai'
  content?: string
  sql?: string
  explanation?: string
  error?: string
  loading?: boolean
  timestamp: number
}

const aiStore = useAIStore()
// const settingsStore = useSettingsStore()

// 主题和语言同步
// const { config, theme, language, isDark, locale } = useSync()

// 国际化
const { t } = useI18n()

// 降级处理 - 使用AI store中的状态
const isInFallbackMode = computed(() => aiStore.isInFallbackMode)
const fallbackStatus = computed(() => aiStore.getFallbackStatus())

const queryInput = ref('')
const chatMessages = ref<ChatMessage[]>([])
const messagesContainer = ref<HTMLElement>()

// 快速示例
const quickExamples = ref([
  {
    id: 1,
    title: '查询用户信息',
    query: '查询所有活跃用户的基本信息'
  },
  {
    id: 2,
    title: '统计订单数量',
    query: '统计每个月的订单总数'
  },
  {
    id: 3,
    title: '查询热门商品',
    query: '查询销量前10的商品'
  },
  {
    id: 4,
    title: '用户行为分析',
    query: '分析用户的购买行为模式'
  }
])

// 生成消息ID
const generateMessageId = () => {
  return Date.now().toString() + Math.random().toString(36).substr(2, 9)
}

// 格式化时间
const formatTime = (timestamp: number) => {
  const date = new Date(timestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  
  if (diff < 60000) { // 1分钟内
    return '刚刚'
  } else if (diff < 3600000) { // 1小时内
    return `${Math.floor(diff / 60000)}分钟前`
  } else if (date.toDateString() === now.toDateString()) { // 今天
    return date.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })
  } else {
    return date.toLocaleDateString('zh-CN', { month: 'short', day: 'numeric', hour: '2-digit', minute: '2-digit' })
  }
}

// 滚动到底部
const scrollToBottom = async () => {
  await nextTick()
  if (messagesContainer.value) {
    messagesContainer.value.scrollTop = messagesContainer.value.scrollHeight
  }
}

// 添加用户消息
const addUserMessage = (content: string) => {
  const message: ChatMessage = {
    id: generateMessageId(),
    type: 'user',
    content,
    timestamp: Date.now()
  }
  chatMessages.value.push(message)
  scrollToBottom()
  return message.id
}

// 添加AI消息（加载状态）
const addAIMessage = (loading = false) => {
  const message: ChatMessage = {
    id: generateMessageId(),
    type: 'ai',
    loading,
    timestamp: Date.now()
  }
  chatMessages.value.push(message)
  scrollToBottom()
  return message.id
}

// 更新AI消息
const updateAIMessage = (messageId: string, updates: Partial<ChatMessage>) => {
  const messageIndex = chatMessages.value.findIndex(m => m.id === messageId)
  if (messageIndex !== -1) {
    chatMessages.value[messageIndex] = { ...chatMessages.value[messageIndex], ...updates }
    scrollToBottom()
  }
}

// 生成降级建议消息
const generateFallbackMessage = (suggestion?: { title: string; description: string }) => {
  if (!suggestion) {
    return t('ai.error.connection')
  }
  
  const templates = aiStore.getSQLTemplates()
  const templateText = templates.map(template => `• ${template.title}: ${template.sql}`).join('\n')
  
  return `${suggestion.title}\n\n${suggestion.description}\n\n${t('ai.fallback.suggestion')}:\n${templateText}`
}

// 提交查询
const handleSubmit = async () => {
  if (!queryInput.value.trim()) {
    ElMessage.warning('请输入查询内容')
    return
  }

  const userQuery = queryInput.value.trim()
  queryInput.value = ''
  
  // 添加用户消息
  addUserMessage(userQuery)
  
  // 添加AI加载消息
  const aiMessageId = addAIMessage(true)

  // 调用AI store的convertToSQL方法（已包含fallback处理）
  await aiStore.convertToSQL(userQuery)
  
  // 检查结果并更新消息
  if (aiStore.currentSQL) {
    // 成功生成SQL
    updateAIMessage(aiMessageId, {
      loading: false,
      sql: aiStore.currentSQL,
      explanation: aiStore.currentExplanation
    })
  } else if (aiStore.isInFallbackMode) {
    // 进入降级模式
    const suggestion = aiStore.fallbackSuggestion
    updateAIMessage(aiMessageId, {
      loading: false,
      error: generateFallbackMessage(suggestion || undefined)
    })
  } else if (aiStore.error) {
    // 其他错误
    updateAIMessage(aiMessageId, {
      loading: false,
      error: aiStore.error
    })
  } else {
    // 未知状态
    updateAIMessage(aiMessageId, {
      loading: false,
      error: t('ai.error.unknown')
    })
  }
}

// 处理换行
const handleNewLine = () => {
  queryInput.value += '\n'
}

// 清空对话
const clearChat = () => {
  chatMessages.value = []
  aiStore.clearCurrentResult()
  ElMessage.success(t('ai.message.cleared'))
}

// 重置降级状态
const resetFallbackState = async () => {
  await aiStore.retryConnection()
  ElMessage.info('正在重新连接AI服务...')
}

const useExample = (exampleQuery: string) => {
  queryInput.value = exampleQuery
}

const copySQL = async (sql: string) => {
  try {
    await navigator.clipboard.writeText(sql)
    ElMessage.success(t('ai.message.copied'))
  } catch {
    ElMessage.error(t('ai.message.copyFailed'))
  }
}

const executeSQL = (_sql: string) => {
  // 这里可以触发SQL执行事件，由父组件处理
  ElMessage.info(t('ai.message.executed'))
}

// 生命周期
onMounted(async () => {
  // 检查连接状态
  await aiStore.checkHealth()
  // 加载模型信息
  await aiStore.loadModelInfo()
})
</script>

<style scoped>
.ai-chat-container {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--color-background);
}

.chat-header {
  padding: 1rem 1.5rem;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-background);
  flex-shrink: 0;
}

.chat-messages {
  flex: 1;
  padding: 1rem 1.5rem;
  overflow-y: auto;
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.welcome-message {
  display: flex;
  justify-content: flex-start;
}

.message-wrapper {
  display: flex;
  flex-direction: column;
}

.message-bubble {
  max-width: 80%;
  padding: 0.75rem 1rem;
  border-radius: 1rem;
  position: relative;
  word-wrap: break-word;
}

.user-message {
  align-self: flex-end;
  background: #007AFF;
  color: white;
  border-bottom-right-radius: 0.25rem;
}

.ai-message {
  align-self: flex-start;
  background: var(--color-background-soft, #f8f9fa);
  border: 1px solid var(--color-border);
  border-bottom-left-radius: 0.25rem;
}

.message-content {
  line-height: 1.5;
}

.message-time {
  font-size: 0.75rem;
  color: rgba(255, 255, 255, 0.7);
  margin-top: 0.5rem;
  text-align: right;
}

.ai-message .message-time {
  color: var(--color-text-2, #666);
  text-align: left;
}

.example-item {
  padding: 0.5rem 0;
  border-bottom: 1px solid var(--color-border-soft, #eee);
}

.example-item:last-child {
  border-bottom: none;
}

.loading-dots {
  display: flex;
  gap: 0.25rem;
}

.dot {
  width: 0.5rem;
  height: 0.5rem;
  background: var(--color-text-2, #666);
  border-radius: 50%;
  animation: loading-bounce 1.4s ease-in-out infinite both;
}

.dot:nth-child(1) { animation-delay: -0.32s; }
.dot:nth-child(2) { animation-delay: -0.16s; }
.dot:nth-child(3) { animation-delay: 0s; }

@keyframes loading-bounce {
  0%, 80%, 100% {
    transform: scale(0.8);
    opacity: 0.5;
  }
  40% {
    transform: scale(1);
    opacity: 1;
  }
}

.error-content {
  color: var(--color-danger, #f56565);
}

.success-content {
  color: var(--color-text, #333);
}

.sql-block {
  margin: 0.75rem 0;
  background: var(--color-background-mute, #f5f5f5);
  border: 1px solid var(--color-border);
  border-radius: 0.5rem;
  overflow: hidden;
}

.sql-code {
  background: #1a1a1a;
  color: #00ff00;
  padding: 1rem;
  margin: 0;
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  font-size: 0.875rem;
  line-height: 1.4;
  overflow-x: auto;
  white-space: pre-wrap;
}

.explanation-block {
  margin-top: 0.75rem;
  padding: 0.75rem;
  background: var(--color-background-soft, #f8f9fa);
  border-radius: 0.5rem;
  border-left: 3px solid #007AFF;
}

.chat-input {
  padding: 1rem 1.5rem;
  border-top: 1px solid var(--color-border);
  background: var(--color-background);
  flex-shrink: 0;
}

.input-container {
  display: flex;
  gap: 0.75rem;
  align-items: flex-end;
}

.message-input {
  flex: 1;
}

.send-button {
  flex-shrink: 0;
  height: auto;
  min-height: 2.5rem;
}

.input-hint {
  margin-top: 0.5rem;
  text-align: center;
}

/* 暗黑模式适配 */
html.dark .ai-message {
  background: var(--color-background-soft, #2a2a2a);
  border-color: var(--color-border, #404040);
}

html.dark .sql-code {
  background: #0d1117;
  color: #58a6ff;
}

html.dark .explanation-block {
  background: var(--color-background-soft, #2a2a2a);
}

html.dark .sql-block {
  background: var(--color-background-mute, #1a1a1a);
}

/* 响应式设计 */
@media (max-width: 768px) {
  .chat-header {
    padding: 0.75rem 1rem;
  }
  
  .chat-messages {
    padding: 0.75rem 1rem;
  }
  
  .chat-input {
    padding: 0.75rem 1rem;
  }
  
  .message-bubble {
    max-width: 90%;
    padding: 0.625rem 0.875rem;
  }
  
  .input-container {
    gap: 0.5rem;
  }
}

/* 滚动条样式 */
.chat-messages::-webkit-scrollbar {
  width: 6px;
}

.chat-messages::-webkit-scrollbar-track {
  background: transparent;
}

.chat-messages::-webkit-scrollbar-thumb {
  background: var(--color-border, #ddd);
  border-radius: 3px;
}

.chat-messages::-webkit-scrollbar-thumb:hover {
  background: var(--color-text-2, #999);
}
</style>