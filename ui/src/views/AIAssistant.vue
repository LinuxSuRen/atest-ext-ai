<template>
  <div class="ai-assistant-container">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <div class="title-section">
          <h1 class="text-2xl font-bold text-gray-800 dark:text-gray-200">
            AI SQL 助手
          </h1>
          <p class="text-gray-600 dark:text-gray-400 mt-1">
            使用自然语言生成SQL查询，提高开发效率
          </p>
        </div>
        
        <div class="status-section">
          <el-tag 
            :type="aiStore.isConnected ? 'success' : 'danger'"
            size="large"
          >
            {{ aiStore.isConnected ? '服务已连接' : '服务未连接' }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- 主要内容区域 -->
    <div class="main-content">
      <!-- 左侧面板 - AI聊天和历史 -->
      <div class="left-panel">
        <!-- AI聊天组件 -->
        <div class="chat-section">
          <AIChat @sql-generated="handleSQLGenerated" />
        </div>
        
        <!-- 历史记录面板 -->
        <div class="history-section mt-6">
          <HistoryPanel 
            @load-history="handleLoadHistory"
            @clear-history="handleClearHistory"
          />
        </div>
      </div>

      <!-- 右侧面板 - SQL编辑器和结果显示 -->
      <div class="right-panel">
        <!-- SQL编辑器 -->
        <div class="editor-section">
          <SQLEditor 
            :initial-sql="currentSQL"
            :show-history="false"
            @sql-executed="handleSQLExecuted"
            @sql-changed="handleSQLChanged"
          />
        </div>
        
        <!-- 结果显示 -->
        <div class="result-section mt-6">
          <ResultDisplay 
            :result="executionResult"
            :loading="isExecuting"
            @export-result="handleExportResult"
          />
        </div>
      </div>
    </div>

    <!-- 设置面板 -->
    <div class="settings-section">
      <SettingsPanel 
        v-model:visible="showSettings"
        @settings-changed="handleSettingsChanged"
      />
    </div>

    <!-- 浮动操作按钮 -->
    <div class="floating-actions">
      <el-button 
        type="primary" 
        circle 
        size="large"
        @click="showSettings = true"
      >
        <el-icon><Setting /></el-icon>
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { ElMessage, ElNotification } from 'element-plus'
import { Setting } from '@element-plus/icons-vue'
import { useAIStore } from '@/stores/ai'
import { useSettingsStore } from '@/stores/settings'
import AIChat from '@/components/AIChat.vue'
import SQLEditor from '@/components/SQLEditor.vue'
import ResultDisplay from '@/components/ResultDisplay.vue'
import HistoryPanel from '@/components/HistoryPanel.vue'
import SettingsPanel from '@/components/SettingsPanel.vue'

// Stores
const aiStore = useAIStore()
const settingsStore = useSettingsStore()

// 响应式数据
const showSettings = ref(false)
const currentSQL = ref('')
const executionResult = ref<any>(null)
const isExecuting = ref(false)

// 计算属性
const hasCurrentSQL = computed(() => !!currentSQL.value.trim())

// 方法
const handleSQLGenerated = (data: { sql: string; query: string; explanation?: string }) => {
  currentSQL.value = data.sql
  ElNotification({
    title: 'SQL生成成功',
    message: '已将生成的SQL加载到编辑器中',
    type: 'success',
    duration: 3000
  })
}

const handleLoadHistory = (historyItem: any) => {
  currentSQL.value = historyItem.sql
  ElMessage.success('已加载历史查询')
}

const handleClearHistory = () => {
  aiStore.clearHistory()
  ElMessage.success('历史记录已清空')
}

const handleSQLExecuted = (result: any) => {
  executionResult.value = result
  isExecuting.value = false
  
  if (result.success) {
    ElNotification({
      title: 'SQL执行成功',
      message: `查询完成，返回 ${result.data?.length || 0} 条记录`,
      type: 'success',
      duration: 3000
    })
  } else {
    ElNotification({
      title: 'SQL执行失败',
      message: result.error || '执行过程中发生错误',
      type: 'error',
      duration: 5000
    })
  }
}

const handleSQLChanged = (sql: string) => {
  currentSQL.value = sql
}

const handleExportResult = (format: string) => {
  if (!executionResult.value?.data) {
    ElMessage.warning('没有可导出的数据')
    return
  }
  
  // 这里可以实现不同格式的导出逻辑
  ElMessage.success(`正在导出为 ${format} 格式...`)
}

const handleSettingsChanged = (settings: any) => {
  ElMessage.success('设置已保存')
}

// 监听AI Store的当前SQL变化
watch(
  () => aiStore.currentSQL,
  (newSQL) => {
    if (newSQL && newSQL !== currentSQL.value) {
      currentSQL.value = newSQL
    }
  },
  { immediate: true }
)

// 生命周期
onMounted(async () => {
  // 检查服务连接状态
  try {
    const isHealthy = await aiStore.checkHealth()
    if (isHealthy) {
      ElMessage.success('AI服务连接成功')
      // 加载模型信息
      await aiStore.loadModelInfo()
    } else {
      ElMessage.error('AI服务连接失败，请检查服务状态')
    }
  } catch (error) {
    console.error('Health check failed:', error)
    ElMessage.error('无法连接到AI服务')
  }
})
</script>

<style scoped>
.ai-assistant-container {
  @apply min-h-screen bg-gray-50 dark:bg-gray-900;
}

.page-header {
  @apply bg-white dark:bg-gray-800 shadow-sm border-b border-gray-200 dark:border-gray-700;
}

.header-content {
  @apply max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 flex items-center justify-between;
}

.title-section h1 {
  @apply text-2xl font-bold text-gray-900 dark:text-white;
}

.title-section p {
  @apply text-sm text-gray-600 dark:text-gray-400 mt-1;
}

.main-content {
  @apply max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-8 grid grid-cols-1 lg:grid-cols-2 gap-8;
}

.left-panel {
  @apply space-y-6;
}

.right-panel {
  @apply space-y-6;
}

.chat-section,
.history-section,
.editor-section,
.result-section {
  @apply bg-white dark:bg-gray-800 rounded-lg shadow-sm;
}

.floating-actions {
  @apply fixed bottom-6 right-6 z-50;
}

.floating-actions .el-button {
  @apply shadow-lg;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .main-content {
    @apply grid-cols-1 gap-6;
  }
  
  .left-panel,
  .right-panel {
    @apply space-y-4;
  }
}

@media (max-width: 640px) {
  .header-content {
    @apply flex-col items-start space-y-4;
  }
  
  .main-content {
    @apply px-4 py-6 gap-4;
  }
  
  .floating-actions {
    @apply bottom-4 right-4;
  }
}

/* 深色模式适配 */
@media (prefers-color-scheme: dark) {
  .ai-assistant-container {
    @apply bg-gray-900;
  }
  
  .page-header {
    @apply bg-gray-800 border-gray-700;
  }
  
  .chat-section,
  .history-section,
  .editor-section,
  .result-section {
    @apply bg-gray-800;
  }
}
</style>