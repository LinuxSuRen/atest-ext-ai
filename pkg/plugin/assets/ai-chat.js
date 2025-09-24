(function() {
    console.log('Loading AI Chat Plugin [VERSION 4]...');

    function createAIChatInterface() {
        const container = document.createElement('div');
        container.className = 'ai-chat-container';

        // Create header
        const header = document.createElement('div');
        header.className = 'ai-chat-header';
        header.innerHTML = '<h2><i class="el-icon-chat-dot-round"></i> AI Assistant</h2><p>Natural language to SQL query generator</p>';

        // Create messages container
        const messages = document.createElement('div');
        messages.className = 'ai-chat-messages';
        messages.id = 'ai-chat-messages';

        // Create initial welcome message
        const initialMessage = document.createElement('div');
        initialMessage.className = 'ai-message';
        const messageContent = document.createElement('div');
        messageContent.className = 'ai-message-content';
        messageContent.innerHTML =
            'Hello! I am your AI assistant. I can help you generate SQL queries from natural language descriptions.' +
            '<br><br><strong>Examples:</strong>' +
            '<ul>' +
                '<li>"Show all users created in the last 30 days"</li>' +
                '<li>"Find products with price greater than 100"</li>' +
                '<li>"Count orders by status"</li>' +
            '</ul>';
        initialMessage.appendChild(messageContent);
        messages.appendChild(initialMessage);

        // Create input area
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

        inputContainer.appendChild(textarea);
        inputContainer.appendChild(button);

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

        // Assemble the full interface
        container.appendChild(header);
        container.appendChild(messages);
        container.appendChild(inputArea);

        return container;
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

        // Add loading message
        const loadingMessage = document.createElement('div');
        loadingMessage.className = 'ai-message loading';
        const loadingContent = document.createElement('div');
        loadingContent.className = 'ai-message-content';
        loadingContent.innerHTML = '<i class="el-icon-loading"></i> Generating SQL query...';
        loadingMessage.appendChild(loadingContent);
        messagesContainer.appendChild(loadingMessage);

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
                        prompt: query,
                        config: JSON.stringify({
                            include_explanation: includeExplanation
                        })
                    })
                })
            });

            const result = await response.json();
            messagesContainer.removeChild(loadingMessage);

            if (result.success !== false) {
                let sql = '', meta = '', success = false;
                if (result.data) {
                    for (const pair of result.data) {
                        if (pair.key === 'content') sql = pair.value;
                        if (pair.key === 'meta') meta = pair.value;
                        if (pair.key === 'success') success = pair.value === 'true';
                    }
                }

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

                    const sqlResult = document.createElement('div');
                    sqlResult.className = 'sql-result';

                    const sqlHeader = document.createElement('div');
                    sqlHeader.className = 'sql-header';
                    sqlHeader.innerHTML = '<strong>Generated SQL:</strong><button class="copy-btn" onclick="copyToClipboard(this)">Copy</button>';

                    const sqlCode = document.createElement('pre');
                    sqlCode.className = 'sql-code';
                    sqlCode.textContent = sql;

                    sqlResult.appendChild(sqlHeader);
                    sqlResult.appendChild(sqlCode);

                    if (metaObj.confidence) {
                        const confidence = document.createElement('div');
                        confidence.className = 'confidence';
                        confidence.textContent = 'Confidence: ' + (metaObj.confidence * 100).toFixed(1) + '%';
                        sqlResult.appendChild(confidence);
                    }

                    if (metaObj.model) {
                        const model = document.createElement('div');
                        model.className = 'model';
                        model.textContent = 'Model: ' + metaObj.model;
                        sqlResult.appendChild(model);
                    }

                    aiContent.appendChild(sqlResult);
                    aiMessage.appendChild(aiContent);
                    messagesContainer.appendChild(aiMessage);
                } else {
                    throw new Error('Failed to generate SQL query');
                }
            } else {
                throw new Error(result.message || 'Unknown error occurred');
            }
        } catch (error) {
            console.error('AI Query Error:', error);
            if (messagesContainer.contains(loadingMessage)) {
                messagesContainer.removeChild(loadingMessage);
            }

            const errorMessage = document.createElement('div');
            errorMessage.className = 'ai-message error';
            const errorContent = document.createElement('div');
            errorContent.className = 'ai-message-content';
            errorContent.innerHTML = '<i class="el-icon-warning"></i> Error: ' + error.message + '<br><small>Please try rephrasing your query or check your AI service configuration.</small>';
            errorMessage.appendChild(errorContent);
            messagesContainer.appendChild(errorMessage);
        } finally {
            sendBtn.disabled = false;
            sendBtn.innerHTML = '<i class="el-icon-promotion"></i> Generate SQL';
            messagesContainer.scrollTop = messagesContainer.scrollHeight;
        }
    }

    // Expose plugin interface
    window.ATestPlugin = {
        mount: function(container) {
            console.log('Mounting AI Chat Plugin');
            const chatInterface = createAIChatInterface();
            container.appendChild(chatInterface);

            const sendBtn = document.getElementById('ai-send-btn');
            const input = document.getElementById('ai-input');

            sendBtn.addEventListener('click', handleAIQuery);
            input.addEventListener('keydown', function(e) {
                if (e.key === 'Enter' && e.ctrlKey) {
                    e.preventDefault();
                    handleAIQuery();
                }
            });
            input.focus();
        }
    };

    // Make copyToClipboard globally accessible for onclick handlers
    window.copyToClipboard = copyToClipboard;

    console.log('AI Chat Plugin loaded successfully [VERSION 4]');
})();