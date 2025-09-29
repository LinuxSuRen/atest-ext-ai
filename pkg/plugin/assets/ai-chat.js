(function() {
    // Create conditional logging for production safety
    const debugLog = (typeof process !== 'undefined' && process.env?.NODE_ENV === 'development')
      ? console.log
      : () => {};

    debugLog('Loading AI Chat Plugin [VERSION 6 - Modern Settings]...');
    debugLog('Script execution started at:', new Date().toISOString());

    // Global configuration state
    let currentConfig = {
        language: 'en',
        provider: 'ollama',  // Direct provider selection
        endpoint: 'http://localhost:11434',
        model: '',
        apiKey: '',
        temperature: 0.7,
        maxTokens: 2048,
        status: 'disconnected'
    };

    // Internationalization
    const i18n = {
        en: {
            title: 'AI Assistant',
            subtitle: 'Natural language to SQL query generator',
            settings: 'Settings',
            language: 'Language',
            provider: 'Provider',
            endpoint: 'Endpoint',
            model: 'Model',
            apiKey: 'API Key',
            temperature: 'Temperature',
            maxTokens: 'Max Tokens',
            testConnection: 'Test Connection',
            save: 'Save',
            reset: 'Reset',
            refresh: 'Refresh',
            connected: 'Connected',
            disconnected: 'Disconnected',
            connecting: 'Connecting...',
            optional: '(optional)',
            welcome: 'Welcome to AI Assistant',
            noModelsFound: 'No AI models found',
            quickSetup: 'Quick Setup:',
            installOllama: 'Install Ollama (Recommended)',
            useOnlineAI: 'Use Online AI (API Key needed)',
            learnMore: 'Learn More',
            advanced: 'Advanced',
            examples: 'Examples:',
            localService: 'Local Service',
            onlineService: 'Online Service'
        },
        zh: {
            title: 'AI 助手',
            subtitle: '自然语言转SQL查询生成器',
            settings: '设置',
            language: '语言',
            provider: '提供者',
            endpoint: '端点',
            model: '模型',
            apiKey: 'API密钥',
            temperature: '温度',
            maxTokens: '最大令牌数',
            testConnection: '测试连接',
            save: '保存',
            reset: '重置',
            refresh: '刷新',
            connected: '已连接',
            disconnected: '已断开',
            connecting: '连接中...',
            optional: '(可选)',
            welcome: '欢迎使用AI助手',
            noModelsFound: '未检测到AI模型',
            quickSetup: '快速设置：',
            installOllama: '安装Ollama（推荐）',
            useOnlineAI: '使用在线AI（需要API密钥）',
            learnMore: '了解更多',
            advanced: '高级设置',
            examples: '示例：',
            localService: '本地服务',
            onlineService: '在线服务'
        }
    };

    function t(key) {
        return i18n[currentConfig.language][key] || key;
    }

    // Load configuration from localStorage (provider-specific)
    function loadConfig() {
        console.log('🔍 [DEBUG] Loading configurations for all providers');

        // Load current provider setting
        const globalConfig = localStorage.getItem('atest-ai-global-config');
        if (globalConfig) {
            try {
                const global = JSON.parse(globalConfig);
                currentConfig.provider = global.provider || currentConfig.provider;
                currentConfig.language = global.language || currentConfig.language;
            } catch (e) {
                console.warn('Failed to load global config:', e);
            }
        }

        // Load provider-specific configuration
        const currentProvider = currentConfig.provider;
        const providerConfig = localStorage.getItem(`atest-ai-config-${currentProvider}`);
        if (providerConfig) {
            try {
                const saved = JSON.parse(providerConfig);
                // Only merge provider-specific fields
                currentConfig.endpoint = saved.endpoint || currentConfig.endpoint;
                currentConfig.apiKey = saved.apiKey || currentConfig.apiKey;
                currentConfig.model = saved.model || currentConfig.model;
                currentConfig.temperature = saved.temperature || currentConfig.temperature;
                currentConfig.maxTokens = saved.maxTokens || currentConfig.maxTokens;
                currentConfig.status = saved.status || currentConfig.status;

                console.log(`✅ [DEBUG] Loaded config for ${currentProvider}:`, {
                    endpoint: currentConfig.endpoint,
                    apiKey: currentConfig.apiKey ? 'SET' : 'EMPTY',
                    model: currentConfig.model || 'EMPTY'
                });
            } catch (e) {
                console.warn(`Failed to load config for provider ${currentProvider}:`, e);
            }
        } else {
            console.log(`📋 [DEBUG] No saved config found for ${currentProvider}, using defaults`);
        }
    }

    // Save configuration to localStorage and backend (provider-specific)
    async function saveConfig() {
        console.log('💾 [DEBUG] Saving configuration for provider:', currentConfig.provider);

        // Save global settings (provider, language)
        const globalConfig = {
            provider: currentConfig.provider,
            language: currentConfig.language
        };
        localStorage.setItem('atest-ai-global-config', JSON.stringify(globalConfig));

        // Save provider-specific configuration
        const providerConfig = {
            endpoint: currentConfig.endpoint,
            apiKey: currentConfig.apiKey,
            model: currentConfig.model,
            temperature: currentConfig.temperature,
            maxTokens: currentConfig.maxTokens,
            status: currentConfig.status
        };
        localStorage.setItem(`atest-ai-config-${currentConfig.provider}`, JSON.stringify(providerConfig));

        console.log(`💾 [DEBUG] Saved config for ${currentConfig.provider}:`, {
            endpoint: providerConfig.endpoint,
            apiKey: providerConfig.apiKey ? 'SET' : 'EMPTY',
            model: providerConfig.model || 'EMPTY'
        });

        try {
            // Also save to backend if we have a valid configuration
            // For local providers (ollama), we need endpoint
            // For cloud providers (openai, deepseek), we need apiKey
            const isLocalProvider = currentConfig.provider === 'ollama';
            const hasValidConfig = currentConfig.provider &&
                (isLocalProvider ? currentConfig.endpoint : currentConfig.apiKey);

            if (hasValidConfig) {
                const updateRequest = {
                    provider: currentConfig.provider,
                    config: {
                        provider: currentConfig.provider,
                        endpoint: currentConfig.endpoint,
                        model: currentConfig.model,
                        api_key: currentConfig.apiKey,
                        temperature: currentConfig.temperature,
                        max_tokens: currentConfig.maxTokens
                    }
                };

                console.log('🔥 [DEBUG] Sending update_config request:', updateRequest);
                const response = await fetch('/api/v1/data/query', {
                    method: 'POST',
                    headers: {
                        'Content-Type': 'application/json',
                        'X-Store-Name': 'ai'
                    },
                    body: JSON.stringify({
                        type: 'ai',
                        key: 'update_config',
                        sql: JSON.stringify(updateRequest)
                    })
                });

                console.log('🔥 [DEBUG] Response status:', response.status, response.statusText);
                if (response.ok) {
                    const result = await response.json();
                    console.log('🔥 [DEBUG] Response result:', result);
                    let success = false;
                    if (result.data) {
                        for (const pair of result.data) {
                            if (pair.key === 'success') {
                                success = pair.value === 'true';
                                break;
                            }
                        }
                    }

                    if (success) {
                        showNotification('Configuration saved successfully', 'success');
                    } else {
                        showNotification('Configuration saved locally but backend sync failed', 'warning');
                    }
                } else {
                    showNotification('Configuration saved locally but backend sync failed', 'warning');
                }
            }
        } catch (error) {
            console.error('Failed to sync configuration to backend:', error);
            let errorMsg = 'Configuration saved locally but backend sync failed';

            if (error.message.includes('fetch')) {
                errorMsg += ': AI service not running';
            } else if (error.message.includes('Network')) {
                errorMsg += ': Network error';
            }

            showNotification(errorMsg, 'warning', 4000);
        }

        updateUI();
    }

    function createSettingsPanel() {
        const panel = document.createElement('div');
        panel.className = 'ai-settings-panel';
        panel.id = 'ai-settings-panel';
        panel.style.display = 'none';

        panel.innerHTML = `
            <div class="ai-settings-header">
                <h3><i class="el-icon-setting"></i> <span data-i18n="settings">${t('settings')}</span></h3>
                <button class="ai-settings-close" id="ai-settings-close">
                    <i class="el-icon-close"></i>
                </button>
            </div>
            <div class="ai-settings-content">
                <div class="ai-settings-section">
                    <div class="ai-setting-group">
                        <label data-i18n="language">${t('language')}:</label>
                        <select id="ai-language-select" class="ai-select">
                            <option value="en">English</option>
                            <option value="zh">中文</option>
                        </select>
                    </div>
                </div>

                <div class="ai-settings-section">
                    <h4>🏠 本地服务 (Local Services)</h4>
                    <div class="ai-provider-group">
                        <button class="ai-provider-card active" data-provider="ollama">
                            <i class="ai-provider-icon">🦙</i>
                            <div>
                                <span class="ai-provider-name">Ollama</span>
                                <small class="ai-provider-desc">本地运行的开源模型</small>
                            </div>
                            <div class="ai-status-indicator" id="status-ollama"></div>
                        </button>
                    </div>

                    <h4>☁️ 云端服务 (Cloud Services)</h4>
                    <div class="ai-provider-group">
                        <button class="ai-provider-card" data-provider="openai">
                            <i class="ai-provider-icon">🤖</i>
                            <div>
                                <span class="ai-provider-name">OpenAI</span>
                                <small class="ai-provider-desc">GPT-4, GPT-3.5 系列模型</small>
                            </div>
                            <div class="ai-status-indicator" id="status-openai"></div>
                        </button>

                        <button class="ai-provider-card" data-provider="deepseek">
                            <i class="ai-provider-icon">🧠</i>
                            <div>
                                <span class="ai-provider-name">DeepSeek</span>
                                <small class="ai-provider-desc">DeepSeek Chat & Reasoner</small>
                            </div>
                            <div class="ai-status-indicator" id="status-deepseek"></div>
                        </button>
                    </div>
                </div>

                <div class="ai-settings-section">
                    <div class="ai-setting-group">
                        <label data-i18n="endpoint">${t('endpoint')}:</label>
                        <input type="text" id="ai-endpoint" class="ai-input" placeholder="http://localhost:11434">
                        <small id="ai-endpoint-help" class="ai-help-text">Default Ollama endpoint. Change only if using custom installation.</small>
                    </div>

                    <div class="ai-setting-group">
                        <label data-i18n="model">${t('model')}:</label>
                        <div class="ai-model-selector">
                            <select id="ai-model-select" class="ai-select">
                                <option value="">Select a model...</option>
                            </select>
                            <button id="ai-refresh-models" class="ai-refresh-btn" title="Refresh models">
                                <i class="el-icon-refresh"></i>
                            </button>
                        </div>
                    </div>

                    <div class="ai-setting-group ai-api-key-group" style="display: none;">
                        <label data-i18n="apiKey">${t('apiKey')} <span class="ai-optional" data-i18n="optional">${t('optional')}</span>:</label>
                        <input type="password" id="ai-api-key" class="ai-input" placeholder="sk-...">
                    </div>
                </div>

                <div class="ai-settings-section ai-advanced-section" style="display: none;">
                    <button class="ai-toggle-advanced" id="ai-toggle-advanced">
                        <i class="el-icon-arrow-right"></i>
                        <span data-i18n="advanced">${t('advanced')}</span>
                    </button>
                    <div class="ai-advanced-settings" id="ai-advanced-settings">
                        <div class="ai-setting-group">
                            <label data-i18n="temperature">${t('temperature')}:</label>
                            <div class="ai-slider-group">
                                <input type="range" id="ai-temperature" min="0" max="2" step="0.1" value="0.7" class="ai-slider">
                                <span id="ai-temperature-value">0.7</span>
                            </div>
                        </div>
                        <div class="ai-setting-group">
                            <label data-i18n="maxTokens">${t('maxTokens')}:</label>
                            <input type="number" id="ai-max-tokens" class="ai-input" min="1" max="8192" value="2048">
                        </div>
                    </div>
                </div>

                <div class="ai-settings-actions">
                    <button id="ai-test-connection" class="ai-btn ai-btn-secondary">
                        <i class="el-icon-connection"></i>
                        <span data-i18n="testConnection">${t('testConnection')}</span>
                    </button>
                    <div class="ai-actions-right">
                        <button id="ai-reset-config" class="ai-btn ai-btn-text">
                            <span data-i18n="reset">${t('reset')}</span>
                        </button>
                        <button id="ai-save-config" class="ai-btn ai-btn-primary">
                            <i class="el-icon-check"></i>
                            <span data-i18n="save">${t('save')}</span>
                        </button>
                    </div>
                </div>
            </div>
        `;

        return panel;
    }

    function createWelcomePanel() {
        const panel = document.createElement('div');
        panel.className = 'ai-welcome-panel';
        panel.id = 'ai-welcome-panel';

        panel.innerHTML = `
            <div class="ai-welcome-content">
                <div class="ai-welcome-icon">
                    <i class="el-icon-warning-outline"></i>
                </div>
                <h3 data-i18n="noModelsFound">${t('noModelsFound')}</h3>
                <p data-i18n="quickSetup">${t('quickSetup')}</p>
                <div class="ai-quick-actions">
                    <button class="ai-quick-btn ai-btn-primary" data-action="install-ollama">
                        <i class="el-icon-download"></i>
                        <span data-i18n="installOllama">${t('installOllama')}</span>
                    </button>
                    <button class="ai-quick-btn ai-btn-secondary" data-action="use-online">
                        <i class="el-icon-connection"></i>
                        <span data-i18n="useOnlineAI">${t('useOnlineAI')}</span>
                    </button>
                    <button class="ai-quick-btn ai-btn-text" data-action="learn-more">
                        <i class="el-icon-question"></i>
                        <span data-i18n="learnMore">${t('learnMore')}</span>
                    </button>
                </div>
            </div>
        `;

        return panel;
    }

    function createAIChatInterface() {
        debugLog('Creating AI Chat interface...');
        const container = document.createElement('div');
        container.className = 'ai-chat-container';

        // Create header without settings button
        const header = document.createElement('div');
        header.className = 'ai-chat-header';
        header.innerHTML = `
            <div class="ai-header-content">
                <h2><i class="el-icon-chat-dot-round"></i> <span data-i18n="title">${t('title')}</span></h2>
                <p><span data-i18n="subtitle">${t('subtitle')}</span></p>
            </div>
            <div class="ai-header-actions">
                <div class="ai-status-display" id="ai-status-display">
                    <div class="ai-status-dot disconnected"></div>
                    <span data-i18n="disconnected">${t('disconnected')}</span>
                </div>
            </div>
        `;

        // Create messages container
        const messages = document.createElement('div');
        messages.className = 'ai-chat-messages';
        messages.id = 'ai-chat-messages';

        // Create welcome panel
        const welcomePanel = createWelcomePanel();
        messages.appendChild(welcomePanel);

        // Create initial welcome message
        const initialMessage = document.createElement('div');
        initialMessage.className = 'ai-message';
        initialMessage.id = 'ai-initial-message';
        const messageContent = document.createElement('div');
        messageContent.className = 'ai-message-content';
        messageContent.innerHTML = `
            Hello! I am your AI assistant. I can help you generate SQL queries from natural language descriptions.
            <br><br><strong><span data-i18n="examples">${t('examples')}</span></strong>
            <ul>
                <li>"Show all users created in the last 30 days"</li>
                <li>"Find products with price greater than 100"</li>
                <li>"Count orders by status"</li>
            </ul>
        `;
        initialMessage.appendChild(messageContent);
        messages.appendChild(initialMessage);

        // Create input area with settings button
        const inputArea = document.createElement('div');
        inputArea.className = 'ai-chat-input-area';

        const inputContainer = document.createElement('div');
        inputContainer.className = 'ai-input-container';

        const textarea = document.createElement('textarea');
        textarea.id = 'ai-input';
        textarea.placeholder = 'Describe what you want to query in natural language...';
        textarea.rows = 3;

        const button = document.createElement('button');
        button.id = 'ai-send-btn';
        button.className = 'ai-send-button';
        button.innerHTML = '<i class="el-icon-promotion"></i> Generate SQL';

        // Create settings button for input area
        const settingsButton = document.createElement('button');
        settingsButton.className = 'ai-settings-btn ai-input-settings-btn';
        settingsButton.id = 'ai-settings-btn';
        settingsButton.title = t('settings');
        settingsButton.innerHTML = '<i class="el-icon-setting"></i>';

        inputContainer.appendChild(textarea);
        inputContainer.appendChild(button);
        inputContainer.appendChild(settingsButton);

        const options = document.createElement('div');
        options.className = 'ai-options';
        const label = document.createElement('label');
        const checkbox = document.createElement('input');
        checkbox.type = 'checkbox';
        checkbox.id = 'ai-explain-checkbox';
        checkbox.checked = true;
        label.appendChild(checkbox);
        label.appendChild(document.createTextNode(' Include explanation'));
        options.appendChild(label);

        inputArea.appendChild(inputContainer);
        inputArea.appendChild(options);

        // Create settings panel
        const settingsPanel = createSettingsPanel();

        // Assemble the full interface
        container.appendChild(header);
        container.appendChild(messages);
        container.appendChild(inputArea);
        container.appendChild(settingsPanel);

        return container;
    }

    // Update UI based on current configuration
    function updateUI() {
        // Update language
        const elements = document.querySelectorAll('[data-i18n]');
        elements.forEach(el => {
            const key = el.getAttribute('data-i18n');
            el.textContent = t(key);
        });

        // Update status display
        const statusDisplay = document.getElementById('ai-status-display');
        const statusDot = statusDisplay?.querySelector('.ai-status-dot');
        const statusText = statusDisplay?.querySelector('span');

        if (statusDot && statusText) {
            statusDot.className = `ai-status-dot ${currentConfig.status}`;
            statusText.textContent = t(currentConfig.status);
        }

        // Update form values
        const langSelect = document.getElementById('ai-language-select');
        const endpointInput = document.getElementById('ai-endpoint');
        const modelSelect = document.getElementById('ai-model-select');
        const apiKeyInput = document.getElementById('ai-api-key');
        const temperatureSlider = document.getElementById('ai-temperature');
        const temperatureValue = document.getElementById('ai-temperature-value');
        const maxTokensInput = document.getElementById('ai-max-tokens');

        if (langSelect) langSelect.value = currentConfig.language;
        if (endpointInput) endpointInput.value = currentConfig.endpoint;
        if (modelSelect) modelSelect.value = currentConfig.model;
        if (apiKeyInput) apiKeyInput.value = currentConfig.apiKey;
        if (temperatureSlider) {
            temperatureSlider.value = currentConfig.temperature;
            if (temperatureValue) temperatureValue.textContent = currentConfig.temperature;
        }
        if (maxTokensInput) maxTokensInput.value = currentConfig.maxTokens;

        // Update provider cards
        updateProviderCards();
        updateModelSelection();
    }

    function updateProviderCards() {
        const cards = document.querySelectorAll('.ai-provider-card');
        cards.forEach(card => {
            if (card.getAttribute('data-provider') === currentConfig.provider) {
                card.classList.add('active');
            } else {
                card.classList.remove('active');
            }
        });

        // Show/hide API key field based on provider
        const apiKeyGroup = document.querySelector('.ai-api-key-group');
        if (apiKeyGroup) {
            const isCloudProvider = ['openai', 'deepseek'].includes(currentConfig.provider);
            apiKeyGroup.style.display = isCloudProvider ? 'block' : 'none';
        }

        // Update endpoint placeholder and help text based on provider
        const endpointInput = document.getElementById('ai-endpoint');
        if (endpointInput) {
            const providerConfig = {
                'ollama': {
                    placeholder: 'http://localhost:11434 (Default for Ollama)',
                    helpText: 'Default Ollama endpoint. Change only if using custom installation.',
                    defaultValue: 'http://localhost:11434'
                },
                'openai': {
                    placeholder: 'https://api.openai.com (Leave empty for default)',
                    helpText: 'OpenAI API endpoint. Leave empty to use official API.',
                    defaultValue: ''
                },
                'deepseek': {
                    placeholder: 'https://api.deepseek.com (Leave empty for default)',
                    helpText: 'DeepSeek API endpoint. Leave empty to use official API.',
                    defaultValue: 'https://api.deepseek.com'
                }
            };

            const config = providerConfig[currentConfig.provider];
            if (config) {
                endpointInput.placeholder = config.placeholder;

                // Update endpoint value if switching providers
                if (!currentConfig.endpoint || currentConfig.endpoint === 'http://localhost:11434') {
                    currentConfig.endpoint = config.defaultValue;
                    endpointInput.value = config.defaultValue;
                }

                // Update help text if exists
                const endpointHelp = document.getElementById('ai-endpoint-help');
                if (endpointHelp) {
                    endpointHelp.textContent = config.helpText;
                }
            }
        }
    }

    async function refreshModels() {
        const refreshBtn = document.getElementById('ai-refresh-models');
        const modelSelect = document.getElementById('ai-model-select');

        if (!refreshBtn || !modelSelect) return;

        refreshBtn.disabled = true;
        refreshBtn.innerHTML = '<i class="el-icon-loading"></i>';

        try {
            // Simulated model fetching - replace with actual API call
            const models = await fetchAvailableModels();

            modelSelect.innerHTML = '<option value="">Select a model...</option>';
            models.forEach(model => {
                const option = document.createElement('option');
                option.value = model.id;        // Use model.id for API calls
                option.textContent = `${model.name} (${model.size})`;  // Use model.name for display
                modelSelect.appendChild(option);
            });

            // Auto-select first model if no model is currently configured
            if (!currentConfig.model && models.length > 0) {
                currentConfig.model = models[0].id;  // Use model.id for API calls
                saveConfig();
            }

            if (currentConfig.model) {
                modelSelect.value = currentConfig.model;
            }

        } catch (error) {
            console.error('Failed to fetch models:', error);
            showNotification('Failed to fetch models. Please check your connection.', 'error');
        } finally {
            refreshBtn.disabled = false;
            refreshBtn.innerHTML = '<i class="el-icon-refresh"></i>';
        }
    }

    async function fetchAvailableModels() {
        try {
            // First, get all available providers
            const providersResponse = await fetch('/api/v1/data/query', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Store-Name': 'ai'
                },
                body: JSON.stringify({
                    type: 'ai',
                    key: 'providers',
                    sql: '{}'
                })
            });

            if (!providersResponse.ok) {
                throw new Error(`Providers API error: ${providersResponse.status}`);
            }

            const providersResult = await providersResponse.json();
            let providers = [];

            if (providersResult.data) {
                for (const pair of providersResult.data) {
                    if (pair.key === 'providers') {
                        providers = JSON.parse(pair.value);
                        break;
                    }
                }
            }

            // Find the exact provider by name
            const provider = providers.find(p => p.name === currentConfig.provider);
            if (provider && provider.models && provider.models.length > 0) {
                return provider.models.map(model => ({
                    id: model.id || model.name,          // API name (e.g., "deepseek-chat")
                    name: model.name || model.id,        // Display name (e.g., "DeepSeek Chat")
                    size: model.size || 'Unknown'
                }));
            }

            // If current provider not found or no models, get models for the specific provider
            const modelsResponse = await fetch('/api/v1/data/query', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Store-Name': 'ai'
                },
                body: JSON.stringify({
                    type: 'ai',
                    key: 'models',
                    sql: JSON.stringify({ provider: currentConfig.provider })
                })
            });

            if (!modelsResponse.ok) {
                throw new Error(`Models API error: ${modelsResponse.status}`);
            }

            const modelsResult = await modelsResponse.json();
            if (modelsResult.data) {
                for (const pair of modelsResult.data) {
                    if (pair.key === 'models') {
                        const models = JSON.parse(pair.value);
                        return models.map(model => ({
                            name: model.name || model.id,
                            size: model.size || 'Unknown'
                        }));
                    }
                }
            }

            return [];

        } catch (error) {
            console.error('Error fetching models:', error);

            let errorMessage = 'Failed to fetch models. ';
            if (error.message.includes('fetch')) {
                errorMessage += 'Please check if the AI plugin service is running.';
            } else if (error.message.includes('Network')) {
                errorMessage += 'Please check your network connection.';
            } else {
                errorMessage += 'Please check your AI provider configuration.';
            }
            showNotification(errorMessage, 'error', 5000);

            // Fallback to mock data for development
            const mockModels = {
                'ollama': [
                    { name: 'llama3.2:3b', size: '2GB' },
                    { name: 'gemma2:9b', size: '5GB' },
                    { name: 'codellama:7b', size: '4GB' }
                ],
                'openai': [
                    { name: 'gpt-4o', size: 'Cloud' },
                    { name: 'gpt-4o-mini', size: 'Cloud' },
                    { name: 'gpt-3.5-turbo', size: 'Cloud' }
                ],
                'custom': [
                    { name: 'custom-model-1', size: 'Unknown' }
                ]
            };

            return mockModels[currentConfig.provider] || [];
        }
    }

    function updateModelSelection() {
        refreshModels();
    }

    async function testConnection() {
        const testBtn = document.getElementById('ai-test-connection');
        if (!testBtn) return;

        testBtn.disabled = true;
        testBtn.innerHTML = '<i class="el-icon-loading"></i> Testing...';

        try {
            currentConfig.status = 'connecting';
            updateUI();

            // Simulate connection test - replace with actual API call
            const success = await performConnectionTest();

            if (success) {
                currentConfig.status = 'connected';
                showNotification('Connection successful!', 'success');
            } else {
                throw new Error('Connection failed');
            }

        } catch (error) {
            console.error('Connection test failed:', error);
            currentConfig.status = 'disconnected';
            showNotification('Connection failed. Please check your settings.', 'error');
        } finally {
            updateUI();
            testBtn.disabled = false;
            testBtn.innerHTML = `<i class="el-icon-connection"></i> <span data-i18n="testConnection">${t('testConnection')}</span>`;
        }
    }

    async function performConnectionTest() {
        try {
            // Prepare configuration for testing
            const testConfig = {
                provider: currentConfig.provider,
                endpoint: currentConfig.endpoint || '',
                model: currentConfig.model || '',
                api_key: currentConfig.apiKey || '',
                temperature: currentConfig.temperature || 0.7,
                max_tokens: currentConfig.maxTokens || 2048
            };

            // Call the backend connection test API
            const response = await fetch('/api/v1/data/query', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Store-Name': 'ai'
                },
                body: JSON.stringify({
                    type: 'ai',
                    key: 'test_connection',
                    sql: JSON.stringify(testConfig)
                })
            });

            if (!response.ok) {
                throw new Error(`Connection test API error: ${response.status}`);
            }

            const result = await response.json();
            if (result.data) {
                for (const pair of result.data) {
                    if (pair.key === 'success') {
                        return pair.value === 'true';
                    }
                }
            }

            return false;

        } catch (error) {
            console.error('Connection test error:', error);
            return false;
        }
    }

    function showNotification(message, type = 'info') {
        // Create notification element
        const notification = document.createElement('div');
        notification.className = `ai-notification ai-notification-${type}`;
        notification.innerHTML = `
            <i class="el-icon-${type === 'success' ? 'check' : type === 'error' ? 'warning' : 'info'}"></i>
            <span>${message}</span>
        `;

        document.body.appendChild(notification);

        // Auto remove after 3 seconds
        setTimeout(() => {
            if (document.body.contains(notification)) {
                document.body.removeChild(notification);
            }
        }, 3000);
    }

    function resetConfiguration() {
        currentConfig = {
            language: 'en',
            provider: 'local',  // Updated default to new category
            endpoint: 'http://localhost:11434',
            model: '',
            apiKey: '',
            temperature: 0.7,
            maxTokens: 2048,
            status: 'disconnected'
        };
        saveConfig();
        showNotification('Configuration reset to defaults', 'info');
    }

    function copyToClipboard(button) {
        const sqlCode = button.parentElement.nextElementSibling.textContent;
        navigator.clipboard.writeText(sqlCode).then(function() {
            const originalText = button.textContent;
            button.textContent = 'Copied!';
            setTimeout(function() {
                button.textContent = originalText;
            }, 2000);
        }).catch(function(err) {
            console.error('Failed to copy text: ', err);
        });
    }

    async function handleAIQuery() {
        const input = document.getElementById('ai-input');
        const messagesContainer = document.getElementById('ai-chat-messages');
        const sendBtn = document.getElementById('ai-send-btn');
        const includeExplanation = document.getElementById('ai-explain-checkbox').checked;

        const query = input.value.trim();
        if (!query) return;

        // Add user message
        const userMessage = document.createElement('div');
        userMessage.className = 'user-message';
        const userContent = document.createElement('div');
        userContent.className = 'user-message-content';
        userContent.textContent = query;
        userMessage.appendChild(userContent);
        messagesContainer.appendChild(userMessage);

        // Add loading message with better UX
        const loadingMessage = document.createElement('div');
        loadingMessage.className = 'ai-message loading';
        const loadingContent = document.createElement('div');
        loadingContent.className = 'ai-message-content';

        const loadingSteps = [
            'Analyzing your query...',
            'Understanding database schema...',
            'Generating SQL query...',
            'Validating result...'
        ];

        let currentStep = 0;
        loadingContent.innerHTML = `<i class="el-icon-loading"></i> ${loadingSteps[currentStep]}`;
        loadingMessage.appendChild(loadingContent);
        messagesContainer.appendChild(loadingMessage);

        // Simulate progressive loading steps
        const stepInterval = setInterval(() => {
            currentStep = (currentStep + 1) % loadingSteps.length;
            if (messagesContainer.contains(loadingMessage)) {
                loadingContent.innerHTML = `<i class="el-icon-loading"></i> ${loadingSteps[currentStep]}`;
            } else {
                clearInterval(stepInterval);
            }
        }, 1000);

        // Clear input and update UI
        input.value = '';
        sendBtn.disabled = true;
        sendBtn.innerHTML = '<i class="el-icon-loading"></i> Generating...';
        messagesContainer.scrollTop = messagesContainer.scrollHeight;

        try {
            const response = await fetch('/api/v1/data/query', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Store-Name': 'ai'
                },
                body: JSON.stringify({
                    type: 'ai',
                    key: 'generate',
                    sql: JSON.stringify({
                        model: currentConfig.model || '',
                        prompt: query,
                        config: JSON.stringify({
                            include_explanation: includeExplanation,
                            provider: currentConfig.provider,
                            endpoint: currentConfig.endpoint,
                            api_key: currentConfig.apiKey,
                            temperature: currentConfig.temperature,
                            max_tokens: currentConfig.maxTokens
                        })
                    })
                })
            });

            const result = await response.json();
            clearInterval(stepInterval);  // Clean up interval
            messagesContainer.removeChild(loadingMessage);

            if (result.success !== false && result.data) {
                let sql = '', meta = '', success = false, errorMsg = '';

                // Parse response data
                for (const pair of result.data) {
                    if (pair.key === 'content') sql = pair.value;
                    if (pair.key === 'meta') meta = pair.value;
                    if (pair.key === 'success') success = pair.value === 'true';
                    if (pair.key === 'error') errorMsg = pair.value;
                }

                // Debug logging
                console.log('🔍 [DEBUG] AI Response:', { success, sql, meta, errorMsg });

                if (success && sql) {
                    let metaObj = {};
                    try {
                        metaObj = JSON.parse(meta);
                    } catch (e) {
                        console.warn('Failed to parse meta:', e);
                    }

                    const aiMessage = document.createElement('div');
                    aiMessage.className = 'ai-message';
                    const aiContent = document.createElement('div');
                    aiContent.className = 'ai-message-content';

                    // Add friendly response text first
                    const responseText = document.createElement('div');
                    responseText.className = 'ai-response-text';

                    let responseMessage = "Here's the SQL query I generated for your request:";
                    if (includeExplanation && metaObj.explanation) {
                        responseMessage += `\n\n📝 **Explanation**: ${metaObj.explanation}`;
                    }

                    responseText.innerHTML = responseMessage.replace(/\n/g, '<br>').replace(/\*\*(.*?)\*\*/g, '<strong>$1</strong>');
                    aiContent.appendChild(responseText);

                    // SQL result section
                    const sqlResult = document.createElement('div');
                    sqlResult.className = 'sql-result';

                    const sqlHeader = document.createElement('div');
                    sqlHeader.className = 'sql-header';
                    sqlHeader.innerHTML = '<strong>🔍 Generated SQL:</strong><button class="copy-btn" onclick="copyToClipboard(this)">📋 Copy</button>';

                    const sqlCode = document.createElement('pre');
                    sqlCode.className = 'sql-code';
                    sqlCode.textContent = sql;

                    sqlResult.appendChild(sqlHeader);
                    sqlResult.appendChild(sqlCode);

                    // Metadata section (more user-friendly)
                    if (metaObj.confidence || metaObj.model) {
                        const metaSection = document.createElement('div');
                        metaSection.className = 'ai-meta-section';

                        let metaInfo = [];
                        if (metaObj.confidence) {
                            const confidencePercent = (metaObj.confidence * 100).toFixed(1);
                            let confidenceIcon = '🟢';
                            if (confidencePercent < 70) confidenceIcon = '🟡';
                            if (confidencePercent < 50) confidenceIcon = '🔴';
                            metaInfo.push(`${confidenceIcon} Confidence: ${confidencePercent}%`);
                        }
                        if (metaObj.model) {
                            metaInfo.push(`🤖 Model: ${metaObj.model}`);
                        }

                        metaSection.innerHTML = '<div class="meta-info">' + metaInfo.join(' • ') + '</div>';
                        sqlResult.appendChild(metaSection);
                    }

                    aiContent.appendChild(sqlResult);
                    aiMessage.appendChild(aiContent);
                    messagesContainer.appendChild(aiMessage);
                } else {
                    // Handle case where success is false or no SQL was generated
                    const aiMessage = document.createElement('div');
                    aiMessage.className = 'ai-message error';
                    const aiContent = document.createElement('div');
                    aiContent.className = 'ai-message-content';

                    let displayMessage = errorMsg || 'Failed to generate SQL query';

                    // Check for specific error scenarios
                    if (!success) {
                        displayMessage = errorMsg || 'The AI service encountered an issue. Please check your configuration.';
                    } else if (!sql) {
                        displayMessage = 'No SQL query was generated. Please try rephrasing your request.';
                    }

                    aiContent.innerHTML = `
                        <div class="ai-error-header">
                            <i class="el-icon-warning"></i> ${displayMessage}
                        </div>
                        <div class="ai-error-details">
                            <small>💡 Tip: Make sure your AI service is properly configured and running.</small>
                        </div>
                    `;
                    aiMessage.appendChild(aiContent);
                    messagesContainer.appendChild(aiMessage);
                }
            } else {
                throw new Error(result.message || 'Unknown error occurred');
            }
        } catch (error) {
            console.error('AI Query Error:', error);
            clearInterval(stepInterval);  // Clean up interval
            if (messagesContainer.contains(loadingMessage)) {
                messagesContainer.removeChild(loadingMessage);
            }

            const errorMessage = document.createElement('div');
            errorMessage.className = 'ai-message error';
            const errorContent = document.createElement('div');
            errorContent.className = 'ai-message-content';

            // Provide more helpful error messages
            let errorText = 'Sorry, I encountered an issue while generating your SQL query.';
            let helpText = 'Please try rephrasing your query or check your AI service configuration.';

            if (error.message.includes('fetch')) {
                errorText = 'Unable to connect to the AI service.';
                helpText = 'Please check if the AI plugin service is running and configured properly.';
            } else if (error.message.includes('Network')) {
                errorText = 'Network connection issue detected.';
                helpText = 'Please check your internet connection and try again.';
            } else if (error.message.includes('timeout')) {
                errorText = 'The AI service took too long to respond.';
                helpText = 'The model might be busy. Please try again in a moment.';
            }

            errorContent.innerHTML = `
                <div class="ai-error-header">
                    <i class="el-icon-warning"></i> ${errorText}
                </div>
                <div class="ai-error-details">
                    <small>${helpText}</small>
                    <br><small><strong>Technical details:</strong> ${error.message}</small>
                </div>
            `;
            errorMessage.appendChild(errorContent);
            messagesContainer.appendChild(errorMessage);
        } finally {
            sendBtn.disabled = false;
            sendBtn.innerHTML = '<i class="el-icon-promotion"></i> Generate SQL';
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }
    }

    function setupEventListeners() {
        // Settings button
        const settingsBtn = document.getElementById('ai-settings-btn');
        const settingsPanel = document.getElementById('ai-settings-panel');
        const settingsClose = document.getElementById('ai-settings-close');

        settingsBtn?.addEventListener('click', () => {
            settingsPanel.style.display = 'flex';
            updateUI();
        });

        settingsClose?.addEventListener('click', () => {
            settingsPanel.style.display = 'none';
        });

        // Language selection
        const langSelect = document.getElementById('ai-language-select');
        langSelect?.addEventListener('change', (e) => {
            currentConfig.language = e.target.value;
            updateUI();
        });

        // Provider cards
        const providerCards = document.querySelectorAll('.ai-provider-card');
        providerCards.forEach(card => {
            card.addEventListener('click', () => {
                const newProvider = card.getAttribute('data-provider');
                const oldProvider = currentConfig.provider;

                // Only switch if different provider selected
                if (newProvider !== oldProvider) {
                    console.log(`🔄 [DEBUG] Switching from ${oldProvider} to ${newProvider}`);

                    // Save current provider's configuration first
                    if (oldProvider) {
                        const oldProviderConfig = {
                            endpoint: currentConfig.endpoint,
                            apiKey: currentConfig.apiKey,
                            model: currentConfig.model,
                            temperature: currentConfig.temperature,
                            maxTokens: currentConfig.maxTokens,
                            status: currentConfig.status
                        };
                        localStorage.setItem(`atest-ai-config-${oldProvider}`, JSON.stringify(oldProviderConfig));
                        console.log(`💾 [DEBUG] Saved ${oldProvider} config before switching`);
                    }

                    // Define provider-specific default configurations
                    const providerDefaults = {
                        ollama: {
                            endpoint: 'http://localhost:11434',
                            apiKey: '',
                            model: ''
                        },
                        openai: {
                            endpoint: '',
                            apiKey: '',
                            model: ''
                        },
                        deepseek: {
                            endpoint: '',
                            apiKey: '',
                            model: ''
                        }
                    };

                    // Load saved configuration for new provider or use defaults
                    const savedConfig = localStorage.getItem(`atest-ai-config-${newProvider}`);
                    let newConfig;

                    if (savedConfig) {
                        try {
                            newConfig = JSON.parse(savedConfig);
                            console.log(`📂 [DEBUG] Loaded saved config for ${newProvider}`);
                        } catch (e) {
                            newConfig = providerDefaults[newProvider] || {};
                            console.log(`⚠️ [DEBUG] Failed to parse saved config, using defaults for ${newProvider}`);
                        }
                    } else {
                        newConfig = providerDefaults[newProvider] || {};
                        console.log(`📋 [DEBUG] No saved config found, using defaults for ${newProvider}`);
                    }

                    // Apply the configuration
                    currentConfig.endpoint = newConfig.endpoint || '';
                    currentConfig.apiKey = newConfig.apiKey || '';
                    currentConfig.model = newConfig.model || '';
                    currentConfig.temperature = newConfig.temperature || 0.7;
                    currentConfig.maxTokens = newConfig.maxTokens || 2048;
                    currentConfig.status = newConfig.status || 'disconnected';

                    console.log(`✨ [DEBUG] Applied config for ${newProvider}:`, {
                        endpoint: currentConfig.endpoint,
                        apiKey: currentConfig.apiKey ? 'SET' : 'EMPTY',
                        model: currentConfig.model || 'EMPTY'
                    });
                }

                currentConfig.provider = newProvider;
                updateProviderCards();
                updateModelSelection();
                updateUI(); // Refresh UI to reflect cleared fields
            });
        });

        // Configuration inputs
        const endpointInput = document.getElementById('ai-endpoint');
        const modelSelect = document.getElementById('ai-model-select');
        const apiKeyInput = document.getElementById('ai-api-key');
        const temperatureSlider = document.getElementById('ai-temperature');
        const maxTokensInput = document.getElementById('ai-max-tokens');

        endpointInput?.addEventListener('input', (e) => {
            currentConfig.endpoint = e.target.value;
        });

        modelSelect?.addEventListener('change', (e) => {
            currentConfig.model = e.target.value;
        });

        apiKeyInput?.addEventListener('input', (e) => {
            currentConfig.apiKey = e.target.value;
        });

        temperatureSlider?.addEventListener('input', (e) => {
            currentConfig.temperature = parseFloat(e.target.value);
            const valueDisplay = document.getElementById('ai-temperature-value');
            if (valueDisplay) valueDisplay.textContent = e.target.value;
        });

        maxTokensInput?.addEventListener('input', (e) => {
            currentConfig.maxTokens = parseInt(e.target.value);
        });

        // Action buttons
        const refreshBtn = document.getElementById('ai-refresh-models');
        const testBtn = document.getElementById('ai-test-connection');
        const saveBtn = document.getElementById('ai-save-config');
        const resetBtn = document.getElementById('ai-reset-config');

        refreshBtn?.addEventListener('click', refreshModels);
        testBtn?.addEventListener('click', testConnection);
        saveBtn?.addEventListener('click', saveConfig);
        resetBtn?.addEventListener('click', resetConfiguration);

        // Advanced settings toggle
        const advancedToggle = document.getElementById('ai-toggle-advanced');
        const advancedSettings = document.getElementById('ai-advanced-settings');
        const advancedSection = document.querySelector('.ai-advanced-section');

        advancedToggle?.addEventListener('click', () => {
            const isExpanded = advancedSettings.style.display === 'block';
            advancedSettings.style.display = isExpanded ? 'none' : 'block';
            const icon = advancedToggle.querySelector('i');
            icon.className = isExpanded ? 'el-icon-arrow-right' : 'el-icon-arrow-down';
        });

        // Welcome panel actions
        const quickBtns = document.querySelectorAll('.ai-quick-btn');
        quickBtns.forEach(btn => {
            btn.addEventListener('click', (e) => {
                const action = e.currentTarget.getAttribute('data-action');
                handleQuickAction(action);
            });
        });

        // Close settings when clicking outside
        settingsPanel?.addEventListener('click', (e) => {
            if (e.target === settingsPanel) {
                settingsPanel.style.display = 'none';
            }
        });

        // Chat functionality
        const sendBtn = document.getElementById('ai-send-btn');
        const input = document.getElementById('ai-input');

        sendBtn?.addEventListener('click', handleAIQuery);
        input?.addEventListener('keydown', function(e) {
            if (e.key === 'Enter' && e.ctrlKey) {
                e.preventDefault();
                handleAIQuery();
            }
        });
    }

    function handleQuickAction(action) {
        switch (action) {
            case 'install-ollama':
                window.open('https://ollama.com/download', '_blank');
                break;
            case 'use-online':
                currentConfig.provider = 'openai';  // Default to OpenAI for cloud service
                const settingsPanel = document.getElementById('ai-settings-panel');
                settingsPanel.style.display = 'flex';
                updateUI();
                break;
            case 'learn-more':
                showNotification('Documentation will be available soon!', 'info');
                break;
        }
    }

    async function checkInitialSetup() {
        const welcomePanel = document.getElementById('ai-welcome-panel');
        const initialMessage = document.getElementById('ai-initial-message');

        // Auto-discover providers on initial setup
        try {
            await discoverProviders();
        } catch (error) {
            console.warn('Failed to auto-discover providers:', error);
        }

        // Check if we have a configured model
        if (!currentConfig.model || currentConfig.status === 'disconnected') {
            welcomePanel.style.display = 'flex';
            initialMessage.style.display = 'none';
        } else {
            welcomePanel.style.display = 'none';
            initialMessage.style.display = 'block';
        }
    }

    async function discoverProviders() {
        try {
            const response = await fetch('/api/v1/data/query', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    'X-Store-Name': 'ai'
                },
                body: JSON.stringify({
                    type: 'ai',
                    key: 'providers',
                    sql: '{}'
                })
            });

            if (!response.ok) {
                console.warn('Failed to discover providers: API error');
                return;
            }

            const result = await response.json();
            let providers = [];

            if (result.data) {
                for (const pair of result.data) {
                    if (pair.key === 'providers') {
                        providers = JSON.parse(pair.value);
                        break;
                    }
                }
            }

            // Update provider status indicators
            updateProviderStatusIndicators(providers);

            // Auto-select first available provider if none is selected
            if (!currentConfig.provider || !['ollama', 'openai', 'deepseek'].includes(currentConfig.provider)) {
                const availableProvider = providers.find(p => p.available && p.models && p.models.length > 0);
                if (availableProvider) {
                    // Use direct provider name
                    currentConfig.provider = availableProvider.name;
                    currentConfig.endpoint = availableProvider.endpoint || '';

                    // Auto-select first model
                    if (availableProvider.models.length > 0 && !currentConfig.model) {
                        currentConfig.model = availableProvider.models[0].name || availableProvider.models[0].id;
                    }

                    currentConfig.status = 'connected';
                    saveConfig();
                }
            }

        } catch (error) {
            console.error('Error discovering providers:', error);
        }
    }

    function updateProviderStatusIndicators(providers) {
        providers.forEach(provider => {
            const indicator = document.getElementById(`status-${provider.name}`);
            if (indicator) {
                indicator.className = 'ai-status-indicator';
                if (provider.available) {
                    indicator.classList.add('connected');
                    indicator.title = `${provider.name}: ${provider.models?.length || 0} models available`;
                } else {
                    indicator.classList.add('disconnected');
                    indicator.title = `${provider.name}: Not available`;
                }
            }
        });
    }

    // Expose plugin interface
    debugLog('Defining window.ATestPlugin interface...');
    window.ATestPlugin = {
        mount: function(container) {
            debugLog('Mounting AI Chat Plugin to container:', container);
            if (!container) {
                console.error('Mount failed: container is null or undefined');
                return;
            }

            // Load configuration first
            loadConfig();

            // Update UI with loaded configuration
            console.log('📋 [DEBUG] Initial config loaded:', {
                provider: currentConfig.provider,
                endpoint: currentConfig.endpoint,
                apiKey: currentConfig.apiKey ? 'SET' : 'EMPTY',
                model: currentConfig.model || 'EMPTY'
            });

            const chatInterface = createAIChatInterface();
            container.appendChild(chatInterface);

            // Setup event listeners
            setupEventListeners();

            // Update UI and check setup
            updateUI();
            checkInitialSetup();

            // Auto-discover providers and refresh models on mount
            setTimeout(async () => {
                await refreshModels();
                // Show/hide welcome panel based on model availability
                const welcomePanel = document.getElementById('ai-welcome-panel');
                if (welcomePanel) {
                    if (currentConfig.model) {
                        welcomePanel.style.display = 'none';
                    } else {
                        welcomePanel.style.display = 'flex';
                    }
                }
            }, 500);

            const input = document.getElementById('ai-input');
            input?.focus();
        }
    };

    // Make copyToClipboard globally accessible for onclick handlers
    window.copyToClipboard = copyToClipboard;

    debugLog('AI Chat Plugin loaded successfully [VERSION 6 - Modern Settings]');
    debugLog('window.ATestPlugin defined:', typeof window.ATestPlugin);
    debugLog('window.ATestPlugin.mount defined:', typeof window.ATestPlugin.mount);
    debugLog('Script setup completed at:', new Date().toISOString());
})();