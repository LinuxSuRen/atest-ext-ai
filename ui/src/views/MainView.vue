<template>
  <div class="main-view-container">
    <!-- 顶部导航栏 -->
    <header class="app-header">
      <div class="header-content">
        <div class="flex items-center space-x-4">
          <div class="logo-section">
            <h1 class="text-xl font-bold text-gray-800">AI SQL Assistant</h1>
            <div class="text-xs text-gray-500">智能SQL查询助手</div>
          </div>
        </div>
        
        <div class="header-actions flex items-center space-x-3">
          <!-- 连接状态 -->
          <div class="connection-status flex items-center space-x-2">
            <el-icon 
              :class="connectionStatusClass"
              class="text-sm"
            >
              <CircleCheckFilled v-if="isConnected" />
              <CircleCloseFilled v-else />
            </el-icon>
            <span class="text-sm text-gray-600">
              {{ connectionStatusText }}
            </span>
          </div>
          
          <!-- 主题切换 -->
          <el-button 
            type="text" 
            size="small"
            @click="toggleTheme"
          >
            <el-icon><Sunny v-if="isDarkMode" /><Moon v-else /></el-icon>
          </el-button>
          
          <!-- 设置按钮 -->
          <el-button 
            type="text" 
            size="small"
            @click="showSettings = true"
          >
            <el-icon><Setting /></el-icon>
          </el-button>
        </div>
      </div>
    </header>

    <!-- 主要内容区域 -->
    <main class="main-content">
      <div class="content-wrapper">
        <!-- 左侧面板 -->
        <aside class="left-panel">
          <el-tabs v-model="leftActiveTab" class="left-tabs">
            <!-- AI聊天面板 -->
            <el-tab-pane label="AI助手" name="chat">
              <div class="panel-content">
                <AIChat 
                  @query-submitted="handleQuerySubmitted"
                  @sql-generated="handleSQLGenerated"
                />
              </div>
            </el-tab-pane>
            
            <!-- 历史记录面板 -->
            <el-tab-pane label="历史记录" name="history">
              <div class="panel-content">
                <HistoryPanel 
                  @reuse-query="handleReuseQuery"
                  @select-item="handleSelectHistoryItem"
                />
              </div>
            </el-tab-pane>
          </el-tabs>
        </aside>

        <!-- 右侧面板 -->
        <section class="right-panel">
          <el-tabs v-model="rightActiveTab" class="right-tabs">
            <!-- SQL编辑器 -->
            <el-tab-pane label="SQL编辑器" name="editor">
              <div class="panel-content">
                <SQLEditor 
                  :initial-sql="currentSQL"
                  @sql-executed="handleSQLExecuted"
                  @sql-changed="handleSQLChanged"
                />
              </div>
            </el-tab-pane>
            
            <!-- 查询结果 -->
            <el-tab-pane label="查询结果" name="results">
              <div class="panel-content">
                <ResultDisplay 
                  :result-data="queryResult"
                  :is-loading="isExecuting"
                  :error="executionError"
                />
              </div>
            </el-tab-pane>
          </el-tabs>
        </section>
      </div>
    </main>

    <!-- 底部状态栏 -->
    <footer class="app-footer">
      <div class="footer-content">
        <div class="status-info flex items-center space-x-6 text-sm text-gray-600">
          <div class="flex items-center space-x-2">
            <el-icon><Clock /></el-icon>
            <span>历史记录: {{ aiStore.queryHistory.length }} 条</span>
          </div>
          
          <div v-if="lastQueryTime" class="flex items-center space-x-2">
            <el-icon><Timer /></el-icon>
            <span>上次查询: {{ formatTime(lastQueryTime) }}</span>
          </div>
          
          <div v-if="currentSQL" class="flex items-center space-x-2">
            <el-icon><Document /></el-icon>
            <span>SQL长度: {{ currentSQL.length }} 字符</span>
          </div>
        </div>
        
        <div class="footer-actions">
          <el-button 
            type="text" 
            size="small"
            @click="clearAll"
          >
            清空所有
          </el-button>
        </div>
      </div>
    </footer>

    <!-- 设置对话框 -->
    <el-drawer 
      v-model="showSettings" 
      title="设置" 
      size="600px"
      direction="rtl"
    >
      <SettingsPanel />
    </el-drawer>

    <!-- 快捷键提示 -->
    <el-dialog 
      v-model="showShortcuts" 
      title="快捷键" 
      width="500px"
    >
      <div class="shortcuts-content">
        <div class="space-y-3">
          <div class="shortcut-item flex justify-between">
            <span>提交查询</span>
            <el-tag size="small">Ctrl + Enter</el-tag>
          </div>
          <div class="shortcut-item flex justify-between">
            <span>格式化SQL</span>
            <el-tag size="small">Ctrl + Shift + F</el-tag>
          </div>
          <div class="shortcut-item flex justify-between">
            <span>清空编辑器</span>
            <el-tag size="small">Ctrl + K</el-tag>
          </div>
          <div class="shortcut-item flex justify-between">
            <span>打开设置</span>
            <el-tag size="small">Ctrl + ,</el-tag>
          </div>
          <div class="shortcut-item flex justify-between">
            <span>显示快捷键</span>
            <el-tag size="small">Ctrl + ?</el-tag>
          </div>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  CircleCheckFilled, 
  CircleCloseFilled, 
  Sunny, 
  Moon, 
  Setting, 
  Clock, 
  Timer, 
  Document 
} from '@element-plus/icons-vue'
import AIChat from '@/components/AIChat.vue'
import SQLEditor from '@/components/SQLEditor.vue'
import ResultDisplay from '@/components/ResultDisplay.vue'
import HistoryPanel from '@/components/HistoryPanel.vue'
import SettingsPanel from '@/components/SettingsPanel.vue'
import { useAIStore, type QueryHistory } from '@/stores/ai'
import { useSettingsStore } from '@/stores/settings'

// Store
const aiStore = useAIStore()
const settingsStore = useSettingsStore()

// 响应式数据
const leftActiveTab = ref('chat')
const rightActiveTab = ref('editor')
const showSettings = ref(false)
const showShortcuts = ref(false)
const isConnected = ref(false)
const isDarkMode = ref(false)
const currentSQL = ref('')
const queryResult = ref<any>(null)
const isExecuting = ref(false)
const executionError = ref<string | null>(null)
const lastQueryTime = ref<Date | null>(null)

// 计算属性
const connectionStatusClass = computed(() => {
  return isConnected.value 
    ? 'text-green-500' 
    : 'text-red-500'
})

const connectionStatusText = computed(() => {
  return isConnected.value ? '已连接' : '未连接'
})

// 方法
const handleQuerySubmitted = (query: string) => {
  console.log('Query submitted:', query)
  lastQueryTime.value = new Date()
}

const handleSQLGenerated = (sql: string) => {
  currentSQL.value = sql
  rightActiveTab.value = 'editor'
  ElMessage.success('SQL已生成，请在编辑器中查看')
}

const handleReuseQuery = (item: QueryHistory) => {
  currentSQL.value = item.sql
  leftActiveTab.value = 'chat'
  rightActiveTab.value = 'editor'
}

const handleSelectHistoryItem = (item: QueryHistory) => {
  console.log('History item selected:', item)
}

const handleSQLExecuted = (result: any) => {
  queryResult.value = result
  rightActiveTab.value = 'results'
  isExecuting.value = false
  executionError.value = null
  ElMessage.success('查询执行完成')
}

const handleSQLChanged = (sql: string) => {
  currentSQL.value = sql
}

const toggleTheme = () => {
  const newTheme = isDarkMode.value ? 'light' : 'dark'
  settingsStore.updateSetting('theme', newTheme)
  isDarkMode.value = !isDarkMode.value
  applyTheme(newTheme)
}

const applyTheme = (theme: string) => {
  const html = document.documentElement
  if (theme === 'dark') {
    html.classList.add('dark')
    isDarkMode.value = true
  } else {
    html.classList.remove('dark')
    isDarkMode.value = false
  }
}

const clearAll = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清空所有内容吗？包括当前SQL和查询结果。',
      '确认清空',
      {
        confirmButtonText: '清空',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    currentSQL.value = ''
    queryResult.value = null
    executionError.value = null
    
    ElMessage.success('已清空所有内容')
  } catch {
    // 用户取消
  }
}

const formatTime = (time: Date) => {
  return time.toLocaleTimeString('zh-CN')
}

const checkConnection = async () => {
  try {
    isConnected.value = await aiStore.checkHealth()
  } catch {
    isConnected.value = false
  }
}

const handleKeyboardShortcuts = (event: KeyboardEvent) => {
  if (event.ctrlKey || event.metaKey) {
    switch (event.key) {
      case ',':
        event.preventDefault()
        showSettings.value = true
        break
      case '?':
        event.preventDefault()
        showShortcuts.value = true
        break
      case 'k':
        event.preventDefault()
        clearAll()
        break
    }
  }
}

// 生命周期
onMounted(async () => {
  // 应用主题
  applyTheme(settingsStore.theme)
  
  // 检查连接状态
  await checkConnection()
  
  // 定期检查连接状态
  const connectionInterval = setInterval(checkConnection, 30000)
  
  // 添加键盘快捷键监听
  document.addEventListener('keydown', handleKeyboardShortcuts)
  
  // 清理函数
  onUnmounted(() => {
    clearInterval(connectionInterval)
    document.removeEventListener('keydown', handleKeyboardShortcuts)
  })
})
</script>

<style scoped>
.main-view-container {
  @apply h-screen flex flex-col bg-gray-50;
}

.app-header {
  @apply bg-white border-b border-gray-200 px-6 py-3;
}

.header-content {
  @apply flex items-center justify-between;
}

.logo-section {
  @apply flex flex-col;
}

.connection-status {
  @apply px-3 py-1 bg-gray-100 rounded-full;
}

.main-content {
  @apply flex-1 overflow-hidden;
}

.content-wrapper {
  @apply h-full flex;
}

.left-panel {
  @apply w-full lg:w-1/2 border-r-0 lg:border-r border-gray-200 bg-white;
}

.right-panel {
  @apply w-full lg:w-1/2 bg-white;
}

.left-tabs,
.right-tabs {
  @apply h-full;
}

.panel-content {
  @apply h-full p-4 overflow-auto;
}

.app-footer {
  @apply bg-white border-t border-gray-200 px-6 py-2;
}

.footer-content {
  @apply flex items-center justify-between;
}

.shortcuts-content {
  @apply space-y-4;
}

.shortcut-item {
  @apply py-2 border-b border-gray-100;
}

/* 深色主题样式 */
:global(.dark) .main-view-container {
  @apply bg-gray-900;
}

:global(.dark) .app-header {
  @apply bg-gray-800 border-gray-700;
}

:global(.dark) .left-panel,
:global(.dark) .right-panel {
  @apply bg-gray-800;
}

:global(.dark) .app-footer {
  @apply bg-gray-800 border-gray-700;
}

:global(.dark) .connection-status {
  @apply bg-gray-700;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .content-wrapper {
    @apply flex-col;
  }
  
  .left-panel {
    @apply w-full border-r-0 border-b border-gray-200 min-h-[50vh];
  }
  
  .right-panel {
    @apply w-full min-h-[50vh];
  }
  
  .header-content {
    @apply flex-col space-y-2;
  }
  
  .logo-section {
    @apply items-center;
  }
}

@media (max-width: 640px) {
  .app-header {
    @apply px-4 py-2;
  }
  
  .panel-content {
    @apply p-2;
  }
  
  .footer-content {
    @apply flex-col space-y-2 text-sm;
  }
}
</style>