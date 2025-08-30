<template>
  <div class="ai-chat-container">
    <div class="chat-header">
      <h2 class="text-xl font-semibold text-gray-800 dark:text-gray-200">
        AI SQL 助手
      </h2>
      <div class="connection-status">
        <el-tag 
          :type="aiStore.isConnected ? 'success' : 'danger'"
          size="small"
        >
          {{ aiStore.isConnected ? '已连接' : '未连接' }}
        </el-tag>
      </div>
    </div>

    <div class="chat-content">
      <!-- 查询输入区域 -->
      <div class="query-input-section">
        <el-card class="mb-4">
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-medium">自然语言查询</span>
              <el-button 
                v-if="aiStore.currentQuery"
                type="text" 
                size="small"
                @click="clearCurrentResult"
              >
                清空
              </el-button>
            </div>
          </template>
          
          <div class="space-y-4">
            <el-input
              v-model="queryInput"
              type="textarea"
              :rows="3"
              placeholder="请输入您的查询需求，例如：查询所有用户的订单信息"
              :disabled="aiStore.isLoading"
              @keydown.ctrl.enter="handleSubmit"
            />
            
            <div class="flex items-center justify-between">
              <div class="text-sm text-gray-500">
                按 Ctrl+Enter 快速提交
              </div>
              <div class="space-x-2">
                <el-button 
                  @click="clearInput"
                  :disabled="aiStore.isLoading || !queryInput.trim()"
                >
                  清空
                </el-button>
                <el-button 
                  type="primary"
                  @click="handleSubmit"
                  :loading="aiStore.isLoading"
                  :disabled="!queryInput.trim()"
                >
                  {{ aiStore.isLoading ? '转换中...' : '转换为SQL' }}
                </el-button>
              </div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 错误提示 -->
      <div v-if="aiStore.error" class="mb-4">
        <el-alert
          :title="aiStore.error"
          type="error"
          show-icon
          closable
          @close="aiStore.clearError"
        />
      </div>

      <!-- 当前结果显示 -->
      <div v-if="aiStore.currentSQL" class="current-result mb-4">
        <el-card>
          <template #header>
            <div class="flex items-center justify-between">
              <span class="font-medium">转换结果</span>
              <div class="space-x-2">
                <el-button 
                  type="text" 
                  size="small"
                  @click="copySQL"
                >
                  复制SQL
                </el-button>
                <el-button 
                  type="text" 
                  size="small"
                  @click="executeSQL"
                >
                  执行查询
                </el-button>
              </div>
            </div>
          </template>
          
          <div class="space-y-4">
            <!-- 原始查询 -->
            <div>
              <div class="text-sm font-medium text-gray-600 mb-2">原始查询：</div>
              <div class="bg-gray-50 p-3 rounded text-sm">
                {{ aiStore.currentQuery }}
              </div>
            </div>
            
            <!-- 生成的SQL -->
            <div>
              <div class="text-sm font-medium text-gray-600 mb-2">生成的SQL：</div>
              <div class="bg-gray-900 text-green-400 p-3 rounded font-mono text-sm overflow-x-auto">
                <pre>{{ aiStore.currentSQL }}</pre>
              </div>
            </div>
            
            <!-- 解释说明 -->
            <div v-if="aiStore.currentExplanation && settingsStore.getSetting('showExplanation')">
              <div class="text-sm font-medium text-gray-600 mb-2">解释说明：</div>
              <div class="bg-blue-50 p-3 rounded text-sm">
                {{ aiStore.currentExplanation }}
              </div>
            </div>
          </div>
        </el-card>
      </div>

      <!-- 快速示例 -->
      <div v-if="!aiStore.currentSQL && !aiStore.isLoading" class="quick-examples">
        <el-card>
          <template #header>
            <span class="font-medium">快速示例</span>
          </template>
          
          <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
            <el-button 
              v-for="example in quickExamples"
              :key="example.id"
              type="text"
              class="text-left p-3 border border-gray-200 rounded hover:bg-gray-50"
              @click="useExample(example.query)"
            >
              <div class="space-y-1">
                <div class="font-medium text-sm">{{ example.title }}</div>
                <div class="text-xs text-gray-500">{{ example.query }}</div>
              </div>
            </el-button>
          </div>
        </el-card>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useAIStore } from '@/stores/ai'
import { useSettingsStore } from '@/stores/settings'

// Store
const aiStore = useAIStore()
const settingsStore = useSettingsStore()

// 响应式数据
const queryInput = ref('')

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

// 方法
const handleSubmit = async () => {
  if (!queryInput.value.trim()) {
    ElMessage.warning('请输入查询内容')
    return
  }

  try {
    await aiStore.convertToSQL(queryInput.value)
    ElMessage.success('转换成功')
  } catch (error) {
    console.error('Conversion failed:', error)
  }
}

const clearInput = () => {
  queryInput.value = ''
}

const clearCurrentResult = () => {
  aiStore.clearCurrentResult()
  queryInput.value = ''
}

const useExample = (exampleQuery: string) => {
  queryInput.value = exampleQuery
}

const copySQL = async () => {
  try {
    await navigator.clipboard.writeText(aiStore.currentSQL)
    ElMessage.success('SQL已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

const executeSQL = () => {
  // 这里可以触发SQL执行事件，由父组件处理
  ElMessage.info('SQL执行功能待实现')
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
  @apply max-w-4xl mx-auto p-4;
}

.chat-header {
  @apply flex items-center justify-between mb-6;
}

.connection-status {
  @apply flex items-center space-x-2;
}

.chat-content {
  @apply space-y-4;
}

.quick-examples .el-button {
  @apply h-auto whitespace-normal;
}

pre {
  @apply whitespace-pre-wrap break-words;
}
</style>