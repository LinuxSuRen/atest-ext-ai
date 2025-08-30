<template>
  <div class="sql-editor-container">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <h3 class="text-lg font-semibold">SQL 编辑器</h3>
            <el-tag v-if="isModified" type="warning" size="small">
              已修改
            </el-tag>
          </div>
          <div class="flex items-center space-x-2">
            <el-button 
              type="text" 
              size="small"
              @click="formatSQL"
              :disabled="!sqlContent.trim()"
            >
              格式化
            </el-button>
            <el-button 
              type="text" 
              size="small"
              @click="clearSQL"
              :disabled="!sqlContent.trim()"
            >
              清空
            </el-button>
            <el-button 
              type="text" 
              size="small"
              @click="copySQL"
              :disabled="!sqlContent.trim()"
            >
              复制
            </el-button>
            <el-button 
              type="primary" 
              size="small"
              @click="executeSQL"
              :disabled="!sqlContent.trim() || isExecuting"
              :loading="isExecuting"
            >
              {{ isExecuting ? '执行中...' : '执行' }}
            </el-button>
          </div>
        </div>
      </template>

      <div class="editor-content">
        <!-- SQL 编辑区域 -->
        <div class="sql-input-section mb-4">
          <el-input
            v-model="sqlContent"
            type="textarea"
            :rows="sqlRows"
            placeholder="请输入或粘贴SQL语句..."
            class="sql-textarea"
            @input="handleSQLChange"
          />
          
          <!-- 行数和字符数统计 -->
          <div class="flex justify-between items-center mt-2 text-sm text-gray-500">
            <div>
              行数: {{ lineCount }} | 字符数: {{ charCount }}
            </div>
            <div class="flex items-center space-x-4">
              <el-button 
                type="text" 
                size="small"
                @click="toggleFullscreen"
              >
                {{ isFullscreen ? '退出全屏' : '全屏编辑' }}
              </el-button>
            </div>
          </div>
        </div>

        <!-- SQL 语法提示 -->
        <div v-if="syntaxError" class="mb-4">
          <el-alert
            :title="syntaxError"
            type="warning"
            show-icon
            closable
            @close="syntaxError = ''"
          />
        </div>

        <!-- 执行结果区域 -->
        <div v-if="executionResult" class="execution-result">
          <div class="result-header mb-3">
            <h4 class="text-md font-medium">执行结果</h4>
            <div class="text-sm text-gray-500">
              执行时间: {{ executionTime }}ms | 
              影响行数: {{ affectedRows }}
            </div>
          </div>
          
          <!-- 成功结果 -->
          <div v-if="executionResult.success" class="success-result">
            <el-alert
              title="执行成功"
              type="success"
              show-icon
              :closable="false"
              class="mb-3"
            />
            
            <!-- 数据表格 -->
            <div v-if="executionResult.data && executionResult.data.length > 0" class="data-table">
              <el-table 
                :data="executionResult.data" 
                border 
                stripe
                max-height="400"
                class="w-full"
              >
                <el-table-column 
                  v-for="column in tableColumns"
                  :key="column"
                  :prop="column"
                  :label="column"
                  :min-width="120"
                  show-overflow-tooltip
                />
              </el-table>
              
              <div class="mt-2 text-sm text-gray-500">
                共 {{ executionResult.data.length }} 条记录
              </div>
            </div>
            
            <!-- 无数据提示 -->
            <div v-else class="no-data text-center py-8 text-gray-500">
              <el-icon class="text-2xl mb-2"><Document /></el-icon>
              <div>查询执行成功，但没有返回数据</div>
            </div>
          </div>
          
          <!-- 错误结果 -->
          <div v-else class="error-result">
            <el-alert
              :title="executionResult.error"
              type="error"
              show-icon
              :closable="false"
            />
          </div>
        </div>

        <!-- SQL 历史记录 -->
        <div v-if="showHistory && sqlHistory.length > 0" class="sql-history mt-6">
          <el-card>
            <template #header>
              <div class="flex items-center justify-between">
                <span class="font-medium">SQL 历史</span>
                <el-button 
                  type="text" 
                  size="small"
                  @click="clearHistory"
                >
                  清空历史
                </el-button>
              </div>
            </template>
            
            <div class="space-y-2 max-h-60 overflow-y-auto">
              <div 
                v-for="(item, index) in sqlHistory"
                :key="index"
                class="history-item p-3 border border-gray-200 rounded cursor-pointer hover:bg-gray-50"
                @click="loadFromHistory(item)"
              >
                <div class="text-sm font-mono text-gray-800 mb-1">
                  {{ item.sql.substring(0, 100) }}{{ item.sql.length > 100 ? '...' : '' }}
                </div>
                <div class="text-xs text-gray-500">
                  {{ formatTime(item.timestamp) }}
                </div>
              </div>
            </div>
          </el-card>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Document } from '@element-plus/icons-vue'

// Props
interface Props {
  initialSQL?: string
  showHistory?: boolean
  readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  initialSQL: '',
  showHistory: true,
  readonly: false
})

// Emits
const emit = defineEmits<{
  sqlChange: [sql: string]
  execute: [sql: string]
}>()

// 响应式数据
const sqlContent = ref(props.initialSQL)
const originalSQL = ref(props.initialSQL)
const isExecuting = ref(false)
const isFullscreen = ref(false)
const syntaxError = ref('')
const executionResult = ref<{ success: boolean; data?: Record<string, unknown>[]; error?: string } | null>(null)
const executionTime = ref(0)
const affectedRows = ref(0)
const sqlHistory = ref<Array<{ sql: string; timestamp: Date }>>([]);

// 计算属性
const isModified = computed(() => sqlContent.value !== originalSQL.value)
const lineCount = computed(() => sqlContent.value.split('\n').length)
const charCount = computed(() => sqlContent.value.length)
const sqlRows = computed(() => {
  const lines = lineCount.value
  return Math.max(6, Math.min(20, lines + 2))
})

const tableColumns = computed(() => {
  if (!executionResult.value?.data || executionResult.value.data.length === 0) {
    return []
  }
  return Object.keys(executionResult.value.data[0] as Record<string, unknown>)
})

// 方法
const handleSQLChange = () => {
  emit('sqlChange', sqlContent.value)
  validateSQL()
}

const validateSQL = () => {
  // 简单的SQL语法检查
  const sql = sqlContent.value.trim().toLowerCase()
  if (sql && !sql.match(/^(select|insert|update|delete|create|drop|alter|show|describe)/)) {
    syntaxError.value = 'SQL语句应该以有效的关键字开头'
  } else {
    syntaxError.value = ''
  }
}

const formatSQL = () => {
  // 简单的SQL格式化
  const formatted = sqlContent.value
    .replace(/\s+/g, ' ')
    .replace(/,/g, ',\n  ')
    .replace(/\bFROM\b/gi, '\nFROM')
    .replace(/\bWHERE\b/gi, '\nWHERE')
    .replace(/\bORDER BY\b/gi, '\nORDER BY')
    .replace(/\bGROUP BY\b/gi, '\nGROUP BY')
    .replace(/\bHAVING\b/gi, '\nHAVING')
    .replace(/\bJOIN\b/gi, '\nJOIN')
    .replace(/\bLEFT JOIN\b/gi, '\nLEFT JOIN')
    .replace(/\bRIGHT JOIN\b/gi, '\nRIGHT JOIN')
    .replace(/\bINNER JOIN\b/gi, '\nINNER JOIN')
  
  sqlContent.value = formatted.trim()
  ElMessage.success('SQL格式化完成')
}

const clearSQL = () => {
  sqlContent.value = ''
  executionResult.value = null
  syntaxError.value = ''
}

const copySQL = async () => {
  try {
    await navigator.clipboard.writeText(sqlContent.value)
    ElMessage.success('SQL已复制到剪贴板')
  } catch {
    ElMessage.error('复制失败')
  }
}

const executeSQL = async () => {
  if (!sqlContent.value.trim()) {
    ElMessage.warning('请输入SQL语句')
    return
  }

  if (syntaxError.value) {
    ElMessage.error('请先修复SQL语法错误')
    return
  }

  isExecuting.value = true
  const startTime = Date.now()
  
  try {
    // 添加到历史记录
    addToHistory(sqlContent.value)
    
    // 触发执行事件
    emit('execute', sqlContent.value)
    
    // 模拟执行结果（实际应该从API获取）
    await new Promise(resolve => setTimeout(resolve, 1000))
    
    executionTime.value = Date.now() - startTime
    affectedRows.value = 0
    
    // 模拟成功结果
    executionResult.value = {
      success: true,
      data: [
        { id: 1, name: '示例数据1', status: '活跃' },
        { id: 2, name: '示例数据2', status: '非活跃' }
      ]
    }
    
    ElMessage.success('SQL执行成功')
  } catch (error) {
    executionResult.value = {
      success: false,
      error: error instanceof Error ? error.message : '执行失败'
    }
    ElMessage.error('SQL执行失败')
  } finally {
    isExecuting.value = false
  }
}

const toggleFullscreen = () => {
  isFullscreen.value = !isFullscreen.value
  // 这里可以实现全屏逻辑
}

const addToHistory = (sql: string) => {
  const historyItem = {
    sql: sql.trim(),
    timestamp: new Date()
  }
  
  // 避免重复
  const exists = sqlHistory.value.some(item => item.sql === sql.trim())
  if (!exists) {
    sqlHistory.value.unshift(historyItem)
    
    // 限制历史记录数量
    if (sqlHistory.value.length > 20) {
      sqlHistory.value = sqlHistory.value.slice(0, 20)
    }
    
    // 保存到本地存储
    localStorage.setItem('sql-history', JSON.stringify(sqlHistory.value))
  }
}

const loadFromHistory = (item: { sql: string; timestamp: Date }) => {
  sqlContent.value = item.sql
  ElMessage.success('已加载历史SQL')
}

const clearHistory = () => {
  sqlHistory.value = []
  localStorage.removeItem('sql-history')
  ElMessage.success('历史记录已清空')
}

const formatTime = (date: Date) => {
  return new Date(date).toLocaleString('zh-CN')
}

// 监听器
watch(() => props.initialSQL, (newSQL) => {
  if (newSQL !== sqlContent.value) {
    sqlContent.value = newSQL
    originalSQL.value = newSQL
  }
})

// 生命周期
onMounted(() => {
  // 加载历史记录
  try {
    const stored = localStorage.getItem('sql-history')
    if (stored) {
      sqlHistory.value = JSON.parse(stored)
    }
  } catch {
    console.error('Failed to load SQL history')
  }
})
</script>

<style scoped>
.sql-editor-container {
  @apply w-full;
}

.sql-textarea :deep(.el-textarea__inner) {
  @apply font-mono text-sm;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
}

.history-item {
  @apply transition-colors duration-200;
}

.data-table {
  @apply border border-gray-200 rounded;
}

.no-data {
  @apply bg-gray-50 rounded;
}
</style>