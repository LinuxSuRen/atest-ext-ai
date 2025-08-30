<template>
  <div class="history-panel-container">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <h3 class="text-lg font-semibold">查询历史</h3>
            <el-tag v-if="aiStore.hasHistory" size="small">
              {{ aiStore.queryHistory.length }} 条记录
            </el-tag>
          </div>
          <div class="flex items-center space-x-2">
            <el-button 
              type="text" 
              size="small"
              @click="toggleView"
            >
              {{ isListView ? '卡片视图' : '列表视图' }}
            </el-button>
            <el-button 
              type="text" 
              size="small"
              @click="exportHistory"
              :disabled="!aiStore.hasHistory"
            >
              导出历史
            </el-button>
            <el-button 
              type="text" 
              size="small"
              @click="clearAllHistory"
              :disabled="!aiStore.hasHistory"
            >
              清空历史
            </el-button>
          </div>
        </div>
      </template>

      <div class="history-content">
        <!-- 搜索和筛选 -->
        <div class="search-filter-section mb-4">
          <div class="flex items-center space-x-3">
            <el-input
              v-model="searchKeyword"
              placeholder="搜索历史记录..."
              style="width: 300px"
              clearable
            >
              <template #prefix>
                <el-icon><Search /></el-icon>
              </template>
            </el-input>
            
            <el-select 
              v-model="dateFilter" 
              placeholder="时间筛选" 
              style="width: 150px"
              clearable
            >
              <el-option label="今天" value="today" />
              <el-option label="本周" value="week" />
              <el-option label="本月" value="month" />
              <el-option label="全部" value="all" />
            </el-select>
            
            <el-select 
              v-model="sortOrder" 
              placeholder="排序方式" 
              style="width: 150px"
            >
              <el-option label="最新优先" value="desc" />
              <el-option label="最旧优先" value="asc" />
            </el-select>
          </div>
        </div>

        <!-- 空状态 -->
        <div v-if="!aiStore.hasHistory" class="empty-state text-center py-12">
          <el-icon class="text-4xl text-gray-400 mb-4"><Clock /></el-icon>
          <div class="text-lg font-medium mb-2 text-gray-600">暂无查询历史</div>
          <div class="text-sm text-gray-500">执行查询后，历史记录将在这里显示</div>
        </div>

        <!-- 历史记录列表 -->
        <div v-else-if="filteredHistory.length === 0" class="no-results text-center py-8">
          <el-icon class="text-3xl text-gray-400 mb-3"><Search /></el-icon>
          <div class="text-lg font-medium text-gray-600 mb-2">没有找到匹配的记录</div>
          <div class="text-sm text-gray-500">尝试调整搜索条件或筛选器</div>
        </div>

        <!-- 列表视图 -->
        <div v-else-if="isListView" class="list-view">
          <div class="space-y-3">
            <div 
              v-for="item in paginatedHistory"
              :key="item.id"
              class="history-item border border-gray-200 rounded-lg p-4 hover:bg-gray-50 cursor-pointer transition-colors"
              @click="selectHistoryItem(item)"
            >
              <div class="flex items-start justify-between">
                <div class="flex-1 min-w-0">
                  <!-- 查询内容 -->
                  <div class="query-content mb-2">
                    <div class="text-sm font-medium text-gray-800 mb-1">
                      {{ item.query }}
                    </div>
                    <div class="text-xs text-gray-500 bg-gray-100 px-2 py-1 rounded font-mono">
                      {{ truncateSQL(item.sql) }}
                    </div>
                  </div>
                  
                  <!-- 元信息 -->
                  <div class="meta-info flex items-center space-x-4 text-xs text-gray-500">
                    <span class="flex items-center">
                      <el-icon class="mr-1"><Clock /></el-icon>
                      {{ formatTime(item.timestamp) }}
                    </span>
                    <span v-if="item.confidence" class="flex items-center">
                      <el-icon class="mr-1"><Star /></el-icon>
                      置信度: {{ Math.round(item.confidence * 100) }}%
                    </span>
                  </div>
                </div>
                
                <!-- 操作按钮 -->
                <div class="actions flex items-center space-x-2 ml-4">
                  <el-button 
                    type="text" 
                    size="small"
                    @click.stop="copySQL(item.sql)"
                  >
                    复制SQL
                  </el-button>
                  <el-button 
                    type="text" 
                    size="small"
                    @click.stop="reuseQuery(item)"
                  >
                    重新使用
                  </el-button>
                  <el-button 
                    type="text" 
                    size="small"
                    @click.stop="removeHistoryItem(item.id)"
                    class="text-red-500 hover:text-red-700"
                  >
                    删除
                  </el-button>
                </div>
              </div>
              
              <!-- 解释说明 -->
              <div v-if="item.explanation && showExplanations" class="explanation mt-3 pt-3 border-t border-gray-100">
                <div class="text-xs text-gray-600">
                  <strong>解释：</strong>{{ item.explanation }}
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 卡片视图 -->
        <div v-else class="card-view">
          <div class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-3 sm:gap-4">
            <el-card 
              v-for="item in paginatedHistory"
              :key="item.id"
              class="history-card cursor-pointer hover:shadow-md transition-shadow min-h-[120px] flex flex-col"
              @click="selectHistoryItem(item)"
            >
              <template #header>
                <div class="flex items-center justify-between flex-shrink-0">
                  <div class="text-sm font-medium text-gray-800 truncate mr-2">
                    {{ item.query }}
                  </div>
                  <el-dropdown trigger="click" @click.stop>
                    <el-icon class="text-gray-400 hover:text-gray-600 flex-shrink-0"><MoreFilled /></el-icon>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item @click="copySQL(item.sql)">
                          复制SQL
                        </el-dropdown-item>
                        <el-dropdown-item @click="reuseQuery(item)">
                          重新使用
                        </el-dropdown-item>
                        <el-dropdown-item @click="removeHistoryItem(item.id)" class="text-red-500">
                          删除
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
              
              <div class="card-content flex-grow">
                <!-- SQL 预览 -->
                <div class="sql-preview mb-3 flex-grow">
                  <div class="bg-gray-900 text-green-400 p-2 rounded text-xs font-mono line-clamp-3">
                    {{ truncateSQL(item.sql, 80) }}
                  </div>
                </div>
                
                <!-- 元信息 -->
                <div class="meta-info space-y-1 flex-shrink-0">
                  <div class="flex items-center text-xs text-gray-500 truncate">
                    <el-icon class="mr-1 flex-shrink-0"><Clock /></el-icon>
                    <span class="truncate">{{ formatTime(item.timestamp) }}</span>
                  </div>
                  <div v-if="item.confidence" class="flex items-center text-xs text-gray-500 truncate">
                    <el-icon class="mr-1 flex-shrink-0"><Star /></el-icon>
                    <span class="truncate">置信度: {{ Math.round(item.confidence * 100) }}%</span>
                  </div>
                </div>
              </div>
            </el-card>
          </div>
        </div>

        <!-- 分页 -->
        <div v-if="filteredHistory.length > pageSize" class="pagination-wrapper mt-6">
          <el-pagination
            v-model:current-page="currentPage"
            v-model:page-size="pageSize"
            :page-sizes="[10, 20, 50]"
            :total="filteredHistory.length"
            layout="total, sizes, prev, pager, next, jumper"
            class="justify-center"
          />
        </div>
      </div>
    </el-card>

    <!-- 详情对话框 -->
    <el-dialog 
      v-model="detailDialogVisible" 
      title="查询详情" 
      width="70%"
      :before-close="closeDetailDialog"
    >
      <div v-if="selectedItem" class="detail-content space-y-4">
        <!-- 原始查询 -->
        <div>
          <h4 class="text-sm font-medium text-gray-700 mb-2">原始查询</h4>
          <div class="bg-gray-50 p-3 rounded text-sm">
            {{ selectedItem.query }}
          </div>
        </div>
        
        <!-- 生成的SQL -->
        <div>
          <h4 class="text-sm font-medium text-gray-700 mb-2">生成的SQL</h4>
          <div class="bg-gray-900 text-green-400 p-3 rounded font-mono text-sm">
            <pre>{{ selectedItem.sql }}</pre>
          </div>
        </div>
        
        <!-- 解释说明 -->
        <div v-if="selectedItem.explanation">
          <h4 class="text-sm font-medium text-gray-700 mb-2">解释说明</h4>
          <div class="bg-blue-50 p-3 rounded text-sm">
            {{ selectedItem.explanation }}
          </div>
        </div>
        
        <!-- 元信息 -->
        <div>
          <h4 class="text-sm font-medium text-gray-700 mb-2">元信息</h4>
          <div class="grid grid-cols-2 gap-4 text-sm">
            <div>
              <span class="text-gray-500">创建时间：</span>
              <span>{{ formatTime(selectedItem.timestamp) }}</span>
            </div>
            <div v-if="selectedItem.confidence">
              <span class="text-gray-500">置信度：</span>
              <span>{{ Math.round(selectedItem.confidence * 100) }}%</span>
            </div>
          </div>
        </div>
      </div>
      
      <template #footer>
        <div class="space-x-2">
          <el-button @click="closeDetailDialog">关闭</el-button>
          <el-button @click="copySQL(selectedItem?.sql || '')">复制SQL</el-button>
          <el-button type="primary" @click="reuseQuery(selectedItem!)">重新使用</el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { 
  Search, 
  Clock, 
  Star, 
  MoreFilled 
} from '@element-plus/icons-vue'
import { useAIStore, type QueryHistory } from '@/stores/ai'
import { useSettingsStore } from '@/stores/settings'

// Emits
const emit = defineEmits<{
  reuseQuery: [item: QueryHistory]
  selectItem: [item: QueryHistory]
}>()

// Store
const aiStore = useAIStore()
const settingsStore = useSettingsStore()

// 响应式数据
const searchKeyword = ref('')
const dateFilter = ref('all')
const sortOrder = ref('desc')
const isListView = ref(true)
const currentPage = ref(1)
const pageSize = ref(10)
const showExplanations = ref(true)
const detailDialogVisible = ref(false)
const selectedItem = ref<QueryHistory | null>(null)

// 计算属性
const filteredHistory = computed(() => {
  let history = [...aiStore.queryHistory]
  
  // 搜索过滤
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    history = history.filter(item => 
      item.query.toLowerCase().includes(keyword) ||
      item.sql.toLowerCase().includes(keyword) ||
      (item.explanation && item.explanation.toLowerCase().includes(keyword))
    )
  }
  
  // 时间过滤
  if (dateFilter.value !== 'all') {
    const now = new Date()
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    
    history = history.filter(item => {
      const itemDate = new Date(item.timestamp)
      
      switch (dateFilter.value) {
        case 'today':
          return itemDate >= today
        case 'week':
          const weekAgo = new Date(today.getTime() - 7 * 24 * 60 * 60 * 1000)
          return itemDate >= weekAgo
        case 'month':
          const monthAgo = new Date(today.getFullYear(), today.getMonth() - 1, today.getDate())
          return itemDate >= monthAgo
        default:
          return true
      }
    })
  }
  
  // 排序
  history.sort((a, b) => {
    const timeA = new Date(a.timestamp).getTime()
    const timeB = new Date(b.timestamp).getTime()
    return sortOrder.value === 'desc' ? timeB - timeA : timeA - timeB
  })
  
  return history
})

const paginatedHistory = computed(() => {
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return filteredHistory.value.slice(start, end)
})

// 方法
const toggleView = () => {
  isListView.value = !isListView.value
}

const selectHistoryItem = (item: QueryHistory) => {
  selectedItem.value = item
  detailDialogVisible.value = true
  emit('selectItem', item)
}

const closeDetailDialog = () => {
  detailDialogVisible.value = false
  selectedItem.value = null
}

const reuseQuery = (item: QueryHistory) => {
  emit('reuseQuery', item)
  ElMessage.success('已加载历史查询')
}

const copySQL = async (sql: string) => {
  try {
    await navigator.clipboard.writeText(sql)
    ElMessage.success('SQL已复制到剪贴板')
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

const removeHistoryItem = async (id: string) => {
  try {
    await ElMessageBox.confirm(
      '确定要删除这条历史记录吗？',
      '确认删除',
      {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    aiStore.removeHistoryItem(id)
    ElMessage.success('历史记录已删除')
  } catch {
    // 用户取消删除
  }
}

const clearAllHistory = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清空所有历史记录吗？此操作不可恢复。',
      '确认清空',
      {
        confirmButtonText: '清空',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    aiStore.clearHistory()
    ElMessage.success('历史记录已清空')
  } catch {
    // 用户取消清空
  }
}

const exportHistory = () => {
  const data = filteredHistory.value.map(item => ({
    查询: item.query,
    SQL: item.sql,
    解释: item.explanation || '',
    时间: formatTime(item.timestamp),
    置信度: item.confidence ? Math.round(item.confidence * 100) + '%' : ''
  }))
  
  const csv = convertToCSV(data)
  downloadCSV(csv, 'query-history.csv')
  ElMessage.success('历史记录已导出')
}

const truncateSQL = (sql: string, maxLength: number = 100) => {
  if (sql.length <= maxLength) return sql
  return sql.substring(0, maxLength) + '...'
}

const formatTime = (timestamp: Date) => {
  return new Date(timestamp).toLocaleString('zh-CN')
}

const convertToCSV = (data: any[]) => {
  if (data.length === 0) return ''
  
  const headers = Object.keys(data[0])
  const csvContent = [
    headers.join(','),
    ...data.map(row => 
      headers.map(header => 
        `"${String(row[header]).replace(/"/g, '""')}"`
      ).join(',')
    )
  ].join('\n')
  
  return csvContent
}

const downloadCSV = (csv: string, filename: string) => {
  const blob = new Blob([csv], { type: 'text/csv;charset=utf-8;' })
  const link = document.createElement('a')
  const url = URL.createObjectURL(blob)
  link.setAttribute('href', url)
  link.setAttribute('download', filename)
  link.style.visibility = 'hidden'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
}
</script>

<style scoped>
.history-panel-container {
  @apply w-full;
}

.history-item {
  @apply transition-all duration-200;
}

.history-item:hover {
  @apply shadow-sm;
}

.history-card {
  @apply h-full;
}

.card-content {
  @apply space-y-3;
}

.sql-preview {
  @apply overflow-hidden;
}

.pagination-wrapper {
  @apply flex justify-center;
}

.detail-content pre {
  @apply whitespace-pre-wrap break-words;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .history-panel-container :deep(.el-card__header) {
    @apply p-3;
  }
  
  .history-panel-container :deep(.el-card__body) {
    @apply p-3;
  }
  
  .card-content {
    @apply space-y-2;
  }
  
  .sql-preview {
    @apply text-xs;
  }
}

@media (max-width: 640px) {
  .history-panel-container :deep(.el-card) {
    @apply min-h-[100px];
  }
  
  .history-panel-container :deep(.el-card__header) {
    @apply p-2 text-sm;
  }
  
  .history-panel-container :deep(.el-card__body) {
    @apply p-2;
  }
  
  .history-panel-container :deep(.el-table) {
    @apply text-xs;
  }
  
  .history-panel-container :deep(.el-table .cell) {
    @apply px-2;
  }
  
  .history-panel-container :deep(.el-pagination) {
    @apply justify-center;
  }
  
  .history-panel-container :deep(.el-pagination .el-pager) {
    @apply text-xs;
  }
  
  .meta-info {
    @apply space-y-1 text-xs;
  }
}

/* 文本截断样式 */
.line-clamp-2 {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}

.line-clamp-3 {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>