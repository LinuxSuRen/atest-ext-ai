<template>
  <div class="result-display-container">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <div class="flex items-center space-x-3">
            <h3 class="text-lg font-semibold">查询结果</h3>
            <el-tag v-if="resultData" :type="resultData.success ? 'success' : 'danger'" size="small">
              {{ resultData.success ? '成功' : '失败' }}
            </el-tag>
          </div>
          <div class="flex items-center space-x-2">
            <el-button 
              v-if="resultData && resultData.success && resultData.data"
              type="text" 
              size="small"
              @click="exportData"
            >
              导出数据
            </el-button>
            <el-button 
              v-if="resultData"
              type="text" 
              size="small"
              @click="clearResult"
            >
              清空结果
            </el-button>
            <el-button 
              type="text" 
              size="small"
              @click="refreshResult"
              :disabled="!canRefresh"
            >
              刷新
            </el-button>
          </div>
        </div>
      </template>

      <div class="result-content">
        <!-- 加载状态 -->
        <div v-if="loading" class="loading-state text-center py-12">
          <el-icon class="text-4xl text-blue-500 mb-4 animate-spin"><Loading /></el-icon>
          <div class="text-lg font-medium mb-2">正在执行查询...</div>
          <div class="text-sm text-gray-500">请稍候</div>
        </div>

        <!-- 空状态 -->
        <div v-else-if="!resultData" class="empty-state text-center py-12">
          <el-icon class="text-4xl text-gray-400 mb-4"><Document /></el-icon>
          <div class="text-lg font-medium mb-2 text-gray-600">暂无查询结果</div>
          <div class="text-sm text-gray-500">执行SQL查询后，结果将在这里显示</div>
        </div>

        <!-- 错误状态 -->
        <div v-else-if="!resultData.success" class="error-state">
          <el-alert
            :title="resultData.error || '查询执行失败'"
            type="error"
            show-icon
            :closable="false"
            class="mb-4"
          />
          
          <div v-if="resultData.details" class="error-details">
            <el-card class="bg-red-50">
              <template #header>
                <span class="text-sm font-medium">错误详情</span>
              </template>
              <pre class="text-sm text-red-700 whitespace-pre-wrap">{{ resultData.details }}</pre>
            </el-card>
          </div>
        </div>

        <!-- 成功状态 -->
        <div v-else class="success-state">
          <!-- 执行信息 -->
          <div class="execution-info mb-4">
            <div class="flex items-center justify-between bg-green-50 p-3 rounded">
              <div class="flex items-center space-x-4">
                <el-icon class="text-green-500"><SuccessFilled /></el-icon>
                <span class="text-sm font-medium text-green-700">查询执行成功</span>
              </div>
              <div class="flex items-center space-x-4 text-sm text-green-600">
                <span>执行时间: {{ executionTime }}ms</span>
                <span>影响行数: {{ affectedRows }}</span>
                <span>返回记录: {{ recordCount }}</span>
              </div>
            </div>
          </div>

          <!-- 数据表格 -->
          <div v-if="resultData.data && resultData.data.length > 0" class="data-section">
            <!-- 表格工具栏 -->
            <div class="table-toolbar mb-3">
              <div class="flex items-center justify-between">
                <div class="flex items-center space-x-3">
                  <span class="text-sm font-medium">数据表格</span>
                  <el-tag size="small">{{ recordCount }} 条记录</el-tag>
                </div>
                <div class="flex items-center space-x-2">
                  <el-input
                    v-model="searchKeyword"
                    placeholder="搜索数据..."
                    size="small"
                    style="width: 200px"
                    clearable
                  >
                    <template #prefix>
                      <el-icon><Search /></el-icon>
                    </template>
                  </el-input>
                  <el-button 
                    type="text" 
                    size="small"
                    @click="toggleTableSize"
                  >
                    {{ isTableCompact ? '展开' : '紧凑' }}
                  </el-button>
                </div>
              </div>
            </div>

            <!-- 数据表格 -->
            <div class="table-container overflow-x-auto">
              <el-table 
                :data="filteredData" 
                border 
                stripe
                :size="isTableCompact ? 'small' : 'default'"
                :max-height="tableMaxHeight"
                class="w-full min-w-[600px]"
                @selection-change="handleSelectionChange"
              >
              <!-- 选择列 -->
              <el-table-column 
                type="selection" 
                width="55"
                :selectable="() => true"
              />
              
              <!-- 序号列 -->
              <el-table-column 
                type="index" 
                label="#" 
                width="60"
                :index="(index: number) => index + 1"
              />
              
              <!-- 数据列 -->
              <el-table-column 
                v-for="column in tableColumns"
                :key="column.key"
                :prop="column.key"
                :label="column.label"
                :min-width="column.width"
                :sortable="column.sortable"
                show-overflow-tooltip
              >
                <template #default="{ row }">
                  <div class="cell-content">
                    <!-- 特殊数据类型处理 -->
                    <span v-if="column.type === 'boolean'" 
                          :class="row[column.key] ? 'text-green-600' : 'text-red-600'">
                      {{ row[column.key] ? '是' : '否' }}
                    </span>
                    <span v-else-if="column.type === 'date'" class="text-gray-700">
                      {{ formatDate(row[column.key]) }}
                    </span>
                    <span v-else-if="column.type === 'number'" class="text-blue-600 font-mono">
                      {{ formatNumber(row[column.key]) }}
                    </span>
                    <span v-else-if="column.type === 'json'" class="text-purple-600">
                      <el-button type="text" size="small" @click="showJSON(row[column.key])">
                        查看JSON
                      </el-button>
                    </span>
                    <span v-else>
                      {{ row[column.key] }}
                    </span>
                  </div>
                </template>
              </el-table-column>
              </el-table>
            </div>

            <!-- 分页 -->
            <div v-if="recordCount > pageSize" class="pagination-wrapper mt-4">
              <el-pagination
                v-model:current-page="currentPage"
                v-model:page-size="pageSize"
                :page-sizes="[10, 20, 50, 100]"
                :total="recordCount"
                layout="total, sizes, prev, pager, next, jumper"
                class="justify-center"
              />
            </div>

            <!-- 选中行操作 -->
            <div v-if="selectedRows.length > 0" class="selected-actions mt-4">
              <el-card class="bg-blue-50">
                <div class="flex items-center justify-between">
                  <span class="text-sm font-medium text-blue-700">
                    已选中 {{ selectedRows.length }} 行数据
                  </span>
                  <div class="space-x-2">
                    <el-button type="text" size="small" @click="exportSelected">
                      导出选中
                    </el-button>
                    <el-button type="text" size="small" @click="clearSelection">
                      取消选择
                    </el-button>
                  </div>
                </div>
              </el-card>
            </div>
          </div>

          <!-- 无数据状态 -->
          <div v-else class="no-data-state text-center py-8">
            <el-icon class="text-3xl text-gray-400 mb-3"><Document /></el-icon>
            <div class="text-lg font-medium text-gray-600 mb-2">查询成功，但没有返回数据</div>
            <div class="text-sm text-gray-500">可能是因为查询条件没有匹配到任何记录</div>
          </div>
        </div>
      </div>
    </el-card>

    <!-- JSON 查看对话框 -->
    <el-dialog v-model="jsonDialogVisible" title="JSON 数据" width="60%">
      <pre class="json-content">{{ jsonContent }}</pre>
      <template #footer>
        <el-button @click="jsonDialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="copyJSON">复制</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { 
  Loading, 
  Document, 
  SuccessFilled, 
  Search 
} from '@element-plus/icons-vue'

// Props
interface Props {
  resultData?: any
  loading?: boolean
  canRefresh?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  loading: false,
  canRefresh: true
})

// Emits
const emit = defineEmits<{
  refresh: []
  clear: []
  export: [data: any[], type: 'all' | 'selected']
}>()

// 响应式数据
const searchKeyword = ref('')
const isTableCompact = ref(false)
const currentPage = ref(1)
const pageSize = ref(20)
const selectedRows = ref<any[]>([])
const jsonDialogVisible = ref(false)
const jsonContent = ref('')
const tableMaxHeight = ref(500)

// 计算属性
const executionTime = computed(() => props.resultData?.executionTime || 0)
const affectedRows = computed(() => props.resultData?.affectedRows || 0)
const recordCount = computed(() => props.resultData?.data?.length || 0)

const tableColumns = computed(() => {
  if (!props.resultData?.data || props.resultData.data.length === 0) {
    return []
  }
  
  const firstRow = props.resultData.data[0]
  return Object.keys(firstRow).map(key => {
    const value = firstRow[key]
    let type = 'string'
    let width = 120
    
    // 推断数据类型
    if (typeof value === 'boolean') {
      type = 'boolean'
      width = 80
    } else if (typeof value === 'number') {
      type = 'number'
      width = 100
    } else if (value instanceof Date || /^\d{4}-\d{2}-\d{2}/.test(value)) {
      type = 'date'
      width = 150
    } else if (typeof value === 'object') {
      type = 'json'
      width = 100
    } else if (typeof value === 'string' && value.length > 50) {
      width = 200
    }
    
    return {
      key,
      label: key.charAt(0).toUpperCase() + key.slice(1),
      type,
      width,
      sortable: type !== 'json'
    }
  })
})

const filteredData = computed(() => {
  if (!props.resultData?.data) return []
  
  let data = props.resultData.data
  
  // 搜索过滤
  if (searchKeyword.value) {
    const keyword = searchKeyword.value.toLowerCase()
    data = data.filter((row: any) => {
      return Object.values(row).some(value => 
        String(value).toLowerCase().includes(keyword)
      )
    })
  }
  
  // 分页
  const start = (currentPage.value - 1) * pageSize.value
  const end = start + pageSize.value
  return data.slice(start, end)
})

// 方法
const clearResult = () => {
  emit('clear')
}

const refreshResult = () => {
  emit('refresh')
}

const exportData = () => {
  if (!props.resultData?.data) return
  emit('export', props.resultData.data, 'all')
}

const exportSelected = () => {
  if (selectedRows.value.length === 0) return
  emit('export', selectedRows.value, 'selected')
}

const toggleTableSize = () => {
  isTableCompact.value = !isTableCompact.value
  tableMaxHeight.value = isTableCompact.value ? 400 : 500
}

const handleSelectionChange = (selection: any[]) => {
  selectedRows.value = selection
}

const clearSelection = () => {
  selectedRows.value = []
}

const showJSON = (data: any) => {
  jsonContent.value = JSON.stringify(data, null, 2)
  jsonDialogVisible.value = true
}

const copyJSON = async () => {
  try {
    await navigator.clipboard.writeText(jsonContent.value)
    ElMessage.success('JSON已复制到剪贴板')
    jsonDialogVisible.value = false
  } catch (error) {
    ElMessage.error('复制失败')
  }
}

const formatDate = (value: any) => {
  if (!value) return '-'
  try {
    return new Date(value).toLocaleString('zh-CN')
  } catch {
    return String(value)
  }
}

const formatNumber = (value: any) => {
  if (typeof value !== 'number') return value
  return value.toLocaleString()
}

// 监听器
watch(() => props.resultData, () => {
  // 重置分页和选择
  currentPage.value = 1
  selectedRows.value = []
  searchKeyword.value = ''
})
</script>

<style scoped>
.result-display-container {
  @apply w-full;
}

.cell-content {
  @apply truncate;
}

.json-content {
  @apply bg-gray-100 p-4 rounded text-sm font-mono max-h-96 overflow-auto;
}

.pagination-wrapper {
  @apply flex justify-center;
}

.loading-state .animate-spin {
  animation: spin 1s linear infinite;
}

.table-container {
  @apply w-full;
}

/* 响应式设计 */
@media (max-width: 1024px) {
  .table-container {
    @apply overflow-x-auto;
  }
  
  .result-display-container :deep(.el-table) {
    min-width: 600px;
  }
  
  .result-display-container :deep(.el-table__header-wrapper) {
    @apply text-xs;
  }
  
  .result-display-container :deep(.el-table__body-wrapper) {
    @apply text-xs;
  }
}

@media (max-width: 640px) {
  .result-display-container :deep(.el-table) {
    min-width: 500px;
  }
  
  .result-display-container :deep(.el-table-column--selection) {
    width: 40px !important;
  }
  
  .result-display-container :deep(.el-pagination) {
    @apply justify-center;
  }
  
  .result-display-container :deep(.el-pagination .el-pager) {
    @apply text-xs;
  }
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}
</style>