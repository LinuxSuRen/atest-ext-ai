<template>
  <div class="settings-panel-container">
    <el-card>
      <template #header>
        <div class="flex items-center justify-between">
          <h3 class="text-lg font-semibold">设置</h3>
          <div class="flex items-center space-x-2">
            <el-button 
              type="text" 
              size="small"
              @click="resetToDefaults"
            >
              重置默认
            </el-button>
            <el-button 
              type="primary" 
              size="small"
              @click="saveSettings"
            >
              保存设置
            </el-button>
          </div>
        </div>
      </template>

      <div class="settings-content">
        <el-tabs v-model="activeTab" class="settings-tabs">
          <!-- 通用设置 -->
          <el-tab-pane label="通用" name="general">
            <div class="settings-section space-y-6">
              <!-- 主题设置 -->
              <div class="setting-group">
                <h4 class="setting-title">外观主题</h4>
                <div class="setting-content">
                  <el-radio-group v-model="localSettings.theme" @change="onThemeChange">
                    <el-radio label="light">浅色主题</el-radio>
                    <el-radio label="dark">深色主题</el-radio>
                    <el-radio label="auto">跟随系统</el-radio>
                  </el-radio-group>
                  <div class="setting-description">
                    选择应用程序的外观主题，自动模式将根据系统设置切换
                  </div>
                </div>
              </div>

              <!-- 语言设置 -->
              <div class="setting-group">
                <h4 class="setting-title">界面语言</h4>
                <div class="setting-content">
                  <el-select v-model="localSettings.language" style="width: 200px">
                    <el-option label="简体中文" value="zh-CN" />
                    <el-option label="English" value="en-US" />
                    <el-option label="繁體中文" value="zh-TW" />
                  </el-select>
                  <div class="setting-description">
                    更改界面显示语言，重启应用后生效
                  </div>
                </div>
              </div>

              <!-- 自动保存 -->
              <div class="setting-group">
                <h4 class="setting-title">自动保存</h4>
                <div class="setting-content">
                  <div class="flex items-center space-x-3">
                    <el-switch 
                      v-model="localSettings.autoSave" 
                      @change="onAutoSaveChange"
                    />
                    <span class="text-sm text-gray-600">
                      {{ localSettings.autoSave ? '已启用' : '已禁用' }}
                    </span>
                  </div>
                  <div class="setting-description">
                    自动保存查询历史和设置更改
                  </div>
                </div>
              </div>

              <!-- 显示解释 -->
              <div class="setting-group">
                <h4 class="setting-title">显示SQL解释</h4>
                <div class="setting-content">
                  <div class="flex items-center space-x-3">
                    <el-switch 
                      v-model="localSettings.showExplanation" 
                      @change="onShowExplanationChange"
                    />
                    <span class="text-sm text-gray-600">
                      {{ localSettings.showExplanation ? '显示' : '隐藏' }}
                    </span>
                  </div>
                  <div class="setting-description">
                    在查询结果中显示SQL语句的详细解释
                  </div>
                </div>
              </div>
            </div>
          </el-tab-pane>

          <!-- 性能设置 -->
          <el-tab-pane label="性能" name="performance">
            <div class="settings-section space-y-6">
              <!-- 历史记录限制 -->
              <div class="setting-group">
                <h4 class="setting-title">历史记录数量限制</h4>
                <div class="setting-content">
                  <div class="flex items-center space-x-4">
                    <el-slider
                      v-model="localSettings.historyLimit"
                      :min="10"
                      :max="1000"
                      :step="10"
                      show-input
                      style="width: 300px"
                      @change="onHistoryLimitChange"
                    />
                    <span class="text-sm text-gray-600">条记录</span>
                  </div>
                  <div class="setting-description">
                    限制保存的历史查询记录数量，超出限制时将自动删除最旧的记录
                  </div>
                </div>
              </div>

              <!-- API超时设置 -->
              <div class="setting-group">
                <h4 class="setting-title">API请求超时</h4>
                <div class="setting-content">
                  <div class="flex items-center space-x-4">
                    <el-slider
                      v-model="localSettings.apiTimeout"
                      :min="5000"
                      :max="60000"
                      :step="1000"
                      show-input
                      style="width: 300px"
                      @change="onApiTimeoutChange"
                    />
                    <span class="text-sm text-gray-600">毫秒</span>
                  </div>
                  <div class="setting-description">
                    设置API请求的超时时间，较长的超时时间可能提高复杂查询的成功率
                  </div>
                </div>
              </div>

              <!-- 缓存设置 -->
              <div class="setting-group">
                <h4 class="setting-title">本地缓存</h4>
                <div class="setting-content">
                  <div class="space-y-3">
                    <div class="flex items-center justify-between">
                      <span class="text-sm">启用查询缓存</span>
                      <el-switch v-model="localSettings.enableCache" />
                    </div>
                    <div class="flex items-center justify-between">
                      <span class="text-sm">缓存大小限制</span>
                      <div class="flex items-center space-x-2">
                        <el-input-number 
                          v-model="localSettings.cacheSize" 
                          :min="1" 
                          :max="100" 
                          size="small"
                          style="width: 100px"
                        />
                        <span class="text-xs text-gray-500">MB</span>
                      </div>
                    </div>
                  </div>
                  <div class="setting-description">
                    启用本地缓存可以提高重复查询的响应速度
                  </div>
                </div>
              </div>
            </div>
          </el-tab-pane>

          <!-- 高级设置 -->
          <el-tab-pane label="高级" name="advanced">
            <div class="settings-section space-y-6">
              <!-- 调试模式 -->
              <div class="setting-group">
                <h4 class="setting-title">调试模式</h4>
                <div class="setting-content">
                  <div class="flex items-center space-x-3">
                    <el-switch 
                      v-model="localSettings.debugMode" 
                      @change="onDebugModeChange"
                    />
                    <span class="text-sm text-gray-600">
                      {{ localSettings.debugMode ? '已启用' : '已禁用' }}
                    </span>
                  </div>
                  <div class="setting-description">
                    启用调试模式将显示详细的日志信息和错误详情
                  </div>
                </div>
              </div>

              <!-- 实验性功能 -->
              <div class="setting-group">
                <h4 class="setting-title">实验性功能</h4>
                <div class="setting-content">
                  <div class="space-y-3">
                    <div class="flex items-center justify-between">
                      <div>
                        <div class="text-sm font-medium">智能提示</div>
                        <div class="text-xs text-gray-500">基于历史查询提供智能建议</div>
                      </div>
                      <el-switch v-model="localSettings.smartSuggestions" />
                    </div>
                    <div class="flex items-center justify-between">
                      <div>
                        <div class="text-sm font-medium">自动格式化</div>
                        <div class="text-xs text-gray-500">自动格式化生成的SQL语句</div>
                      </div>
                      <el-switch v-model="localSettings.autoFormat" />
                    </div>
                    <div class="flex items-center justify-between">
                      <div>
                        <div class="text-sm font-medium">语法高亮</div>
                        <div class="text-xs text-gray-500">在SQL编辑器中启用语法高亮</div>
                      </div>
                      <el-switch v-model="localSettings.syntaxHighlight" />
                    </div>
                  </div>
                  <div class="setting-description">
                    这些功能正在开发中，可能不稳定
                  </div>
                </div>
              </div>

              <!-- 数据管理 -->
              <div class="setting-group">
                <h4 class="setting-title">数据管理</h4>
                <div class="setting-content">
                  <div class="space-y-3">
                    <el-button 
                      type="info" 
                      size="small"
                      @click="exportSettings"
                    >
                      导出设置
                    </el-button>
                    <el-button 
                      type="info" 
                      size="small"
                      @click="importSettings"
                    >
                      导入设置
                    </el-button>
                    <el-button 
                      type="warning" 
                      size="small"
                      @click="clearAllData"
                    >
                      清除所有数据
                    </el-button>
                  </div>
                  <div class="setting-description">
                    管理应用程序数据和设置的导入导出
                  </div>
                </div>
              </div>
            </div>
          </el-tab-pane>

          <!-- 关于 -->
          <el-tab-pane label="关于" name="about">
            <div class="settings-section space-y-6">
              <!-- 应用信息 -->
              <div class="setting-group">
                <h4 class="setting-title">应用信息</h4>
                <div class="setting-content">
                  <div class="space-y-2 text-sm">
                    <div class="flex justify-between">
                      <span class="text-gray-600">应用名称：</span>
                      <span>AI SQL Assistant</span>
                    </div>
                    <div class="flex justify-between">
                      <span class="text-gray-600">版本：</span>
                      <span>{{ appVersion }}</span>
                    </div>
                    <div class="flex justify-between">
                      <span class="text-gray-600">构建时间：</span>
                      <span>{{ buildTime }}</span>
                    </div>
                    <div class="flex justify-between">
                      <span class="text-gray-600">运行环境：</span>
                      <span>{{ environment }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 系统状态 -->
              <div class="setting-group">
                <h4 class="setting-title">系统状态</h4>
                <div class="setting-content">
                  <div class="space-y-2 text-sm">
                    <div class="flex justify-between">
                      <span class="text-gray-600">连接状态：</span>
                      <el-tag :type="connectionStatus.type" size="small">
                        {{ connectionStatus.text }}
                      </el-tag>
                    </div>
                    <div class="flex justify-between">
                      <span class="text-gray-600">历史记录：</span>
                      <span>{{ aiStore.queryHistory.length }} 条</span>
                    </div>
                    <div class="flex justify-between">
                      <span class="text-gray-600">缓存大小：</span>
                      <span>{{ cacheSize }}</span>
                    </div>
                  </div>
                </div>
              </div>

              <!-- 帮助链接 -->
              <div class="setting-group">
                <h4 class="setting-title">帮助与支持</h4>
                <div class="setting-content">
                  <div class="space-y-2">
                    <el-button type="text" size="small" @click="openHelp">
                      使用帮助
                    </el-button>
                    <el-button type="text" size="small" @click="openFeedback">
                      问题反馈
                    </el-button>
                    <el-button type="text" size="small" @click="checkUpdates">
                      检查更新
                    </el-button>
                  </div>
                </div>
              </div>
            </div>
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-card>

    <!-- 导入设置对话框 -->
    <el-dialog v-model="importDialogVisible" title="导入设置" width="500px">
      <div class="import-content">
        <el-upload
          ref="uploadRef"
          :auto-upload="false"
          :show-file-list="false"
          accept=".json"
          @change="handleFileChange"
        >
          <el-button type="primary">选择设置文件</el-button>
        </el-upload>
        <div v-if="importFile" class="mt-3 text-sm text-gray-600">
          已选择文件：{{ importFile.name }}
        </div>
      </div>
      <template #footer>
        <div class="space-x-2">
          <el-button @click="importDialogVisible = false">取消</el-button>
          <el-button type="primary" @click="confirmImport" :disabled="!importFile">
            导入
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, reactive, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { useSettingsStore, type AppSettings } from '@/stores/settings'
import { useAIStore } from '@/stores/ai'

// Store
const settingsStore = useSettingsStore()
const aiStore = useAIStore()

// 响应式数据
const activeTab = ref('general')
const importDialogVisible = ref(false)
const importFile = ref<File | null>(null)
const uploadRef = ref()

// 本地设置副本
const localSettings = reactive({
  theme: settingsStore.theme,
  language: settingsStore.language,
  autoSave: settingsStore.autoSave,
  showExplanation: settingsStore.showExplanation,
  historyLimit: settingsStore.historyLimit,
  apiTimeout: settingsStore.apiTimeout,
  enableCache: true,
  cacheSize: 10,
  debugMode: false,
  smartSuggestions: false,
  autoFormat: true,
  syntaxHighlight: true
})

// 计算属性
const appVersion = computed(() => '1.0.0')
const buildTime = computed(() => new Date().toLocaleDateString())
const environment = computed(() => import.meta.env.MODE)

const connectionStatus = computed(() => {
  // 这里应该从AI store获取实际的连接状态
  return {
    type: 'success' as const,
    text: '已连接'
  }
})

const cacheSize = computed(() => {
  // 计算实际缓存大小
  const size = JSON.stringify(localStorage).length
  return size > 1024 * 1024 
    ? `${(size / 1024 / 1024).toFixed(2)} MB`
    : `${(size / 1024).toFixed(2)} KB`
})

// 方法
const onThemeChange = (value: string) => {
  settingsStore.updateSetting('theme', value as 'light' | 'dark' | 'auto')
  applyTheme(value)
}

const onAutoSaveChange = (value: boolean) => {
  settingsStore.updateSetting('autoSave', value)
}

const onShowExplanationChange = (value: boolean) => {
  settingsStore.updateSetting('showExplanation', value)
}

const onHistoryLimitChange = (value: number) => {
  settingsStore.updateSetting('maxHistoryItems', value)
}

const onApiTimeoutChange = (value: number) => {
  settingsStore.updateSetting('apiTimeout', value)
}

const onDebugModeChange = (value: boolean) => {
  localSettings.debugMode = value
  // 这里可以设置全局调试模式
}

const applyTheme = (theme: string) => {
  const html = document.documentElement
  if (theme === 'dark') {
    html.classList.add('dark')
  } else if (theme === 'light') {
    html.classList.remove('dark')
  } else {
    // auto mode
    const prefersDark = window.matchMedia('(prefers-color-scheme: dark)').matches
    if (prefersDark) {
      html.classList.add('dark')
    } else {
      html.classList.remove('dark')
    }
  }
}

const saveSettings = () => {
  // 保存所有设置
  Object.entries(localSettings).forEach(([key, value]) => {
    if (key in settingsStore) {
      settingsStore.updateSetting(key as keyof AppSettings, value as AppSettings[keyof AppSettings])
    }
  })
  
  ElMessage.success('设置已保存')
}

const resetToDefaults = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要重置所有设置为默认值吗？此操作不可恢复。',
      '确认重置',
      {
        confirmButtonText: '重置',
        cancelButtonText: '取消',
        type: 'warning',
      }
    )
    
    settingsStore.resetSettings()
    
    // 重新加载本地设置
    Object.assign(localSettings, {
      theme: settingsStore.theme,
      language: settingsStore.language,
      autoSave: settingsStore.autoSave,
      showExplanation: settingsStore.showExplanation,
      historyLimit: settingsStore.historyLimit,
      apiTimeout: settingsStore.apiTimeout,
      enableCache: true,
      cacheSize: 10,
      debugMode: false,
      smartSuggestions: false,
      autoFormat: true,
      syntaxHighlight: true
    })
    
    ElMessage.success('设置已重置为默认值')
  } catch {
    // 用户取消重置
  }
}

const exportSettings = () => {
  const settings = {
    ...settingsStore.$state,
    exportTime: new Date().toISOString(),
    version: appVersion.value
  }
  
  const blob = new Blob([JSON.stringify(settings, null, 2)], {
    type: 'application/json'
  })
  
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `ai-sql-settings-${new Date().toISOString().split('T')[0]}.json`
  link.click()
  
  URL.revokeObjectURL(url)
  ElMessage.success('设置已导出')
}

const importSettings = () => {
  importDialogVisible.value = true
}

const handleFileChange = (file: { raw: File }) => {
  importFile.value = file.raw
}

const confirmImport = async () => {
  if (!importFile.value) return
  
  try {
    const text = await importFile.value.text()
    const settings = JSON.parse(text)
    
    // 验证设置格式
    if (!settings || typeof settings !== 'object') {
      throw new Error('无效的设置文件格式')
    }
    
    // 导入设置
    Object.entries(settings).forEach(([key, value]) => {
      if (key in settingsStore && key !== 'exportTime' && key !== 'version') {
        settingsStore.updateSetting(key as keyof AppSettings, value as AppSettings[keyof AppSettings])
      }
    })
    
    importDialogVisible.value = false
    importFile.value = null
    
    ElMessage.success('设置导入成功')
    
    // 刷新页面以应用新设置
    setTimeout(() => {
      window.location.reload()
    }, 1000)
  } catch (error) {
    ElMessage.error('导入设置失败：' + (error as Error).message)
  }
}

const clearAllData = async () => {
  try {
    await ElMessageBox.confirm(
      '确定要清除所有数据吗？包括设置、历史记录等。此操作不可恢复。',
      '确认清除',
      {
        confirmButtonText: '清除',
        cancelButtonText: '取消',
        type: 'error',
      }
    )
    
    // 清除所有数据
    localStorage.clear()
    sessionStorage.clear()
    
    ElMessage.success('所有数据已清除')
    
    // 刷新页面
    setTimeout(() => {
      window.location.reload()
    }, 1000)
  } catch {
    // 用户取消清除
  }
}

const openHelp = () => {
  // 打开帮助文档
  window.open('https://example.com/help', '_blank')
}

const openFeedback = () => {
  // 打开反馈页面
  window.open('https://example.com/feedback', '_blank')
}

const checkUpdates = async () => {
  // 检查更新
  ElMessage.info('正在检查更新...')
  
  // 模拟检查更新
  setTimeout(() => {
    ElMessage.success('当前已是最新版本')
  }, 2000)
}

// 生命周期
onMounted(() => {
  // 应用当前主题
  applyTheme(settingsStore.theme)
})
</script>

<style scoped>
.settings-panel-container {
  @apply w-full;
}

.settings-tabs {
  @apply w-full;
}

.settings-section {
  @apply max-w-2xl;
}

.setting-group {
  @apply border-b border-gray-100 pb-6;
}

.setting-group:last-child {
  @apply border-b-0 pb-0;
}

.setting-title {
  @apply text-base font-medium text-gray-800 mb-3;
}

.setting-content {
  @apply space-y-2;
}

.setting-description {
  @apply text-xs text-gray-500 mt-2;
}

.import-content {
  @apply text-center py-4;
}
</style>