#!/bin/bash
# AI Connection Fix Script for atest-ext-ai

echo "==============================================="
echo "AI Plugin Connection Diagnostic & Fix Script"
echo "==============================================="

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check 1: Ollama Service
echo -e "\n${YELLOW}[1] Checking Ollama Service Status...${NC}"
if pgrep -f "ollama serve" > /dev/null; then
    echo -e "${GREEN}✓ Ollama service is running${NC}"

    # Test Ollama API
    if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
        echo -e "${GREEN}✓ Ollama API is accessible${NC}"

        # List available models
        echo -e "${YELLOW}Available Ollama models:${NC}"
        curl -s http://localhost:11434/api/tags | jq -r '.models[]?.name' 2>/dev/null || echo "No models found"
    else
        echo -e "${RED}✗ Ollama API is not accessible${NC}"
    fi
else
    echo -e "${RED}✗ Ollama service is not running${NC}"
    echo -e "${YELLOW}Starting Ollama service...${NC}"
    if command -v ollama &> /dev/null; then
        ollama serve &
        sleep 3
        echo -e "${GREEN}✓ Ollama service started${NC}"
    else
        echo -e "${RED}✗ Ollama is not installed${NC}"
        echo "Please install Ollama: https://ollama.ai/download"
    fi
fi

# Check 2: Model Availability
echo -e "\n${YELLOW}[2] Checking Model Configuration...${NC}"
CONFIGURED_MODEL="codellama"
AVAILABLE_MODELS=$(curl -s http://localhost:11434/api/tags | jq -r '.models[]?.name' 2>/dev/null)

if [ -z "$AVAILABLE_MODELS" ]; then
    echo -e "${RED}✗ No models available in Ollama${NC}"
    echo -e "${YELLOW}Pulling recommended model (gemma3:1b)...${NC}"
    ollama pull gemma3:1b
    CONFIGURED_MODEL="gemma3:1b"
elif echo "$AVAILABLE_MODELS" | grep -q "$CONFIGURED_MODEL"; then
    echo -e "${GREEN}✓ Configured model '$CONFIGURED_MODEL' is available${NC}"
else
    echo -e "${RED}✗ Configured model '$CONFIGURED_MODEL' is not available${NC}"
    FIRST_MODEL=$(echo "$AVAILABLE_MODELS" | head -1)
    echo -e "${YELLOW}Using available model: $FIRST_MODEL${NC}"
    CONFIGURED_MODEL=$FIRST_MODEL
fi

# Check 3: Plugin Process
echo -e "\n${YELLOW}[3] Checking AI Plugin Process...${NC}"
if pgrep -f "atest-ext-ai" > /dev/null; then
    echo -e "${GREEN}✓ AI Plugin is running${NC}"

    # Show process details
    ps aux | grep -E "atest-ext-ai" | grep -v grep | head -2
else
    echo -e "${RED}✗ AI Plugin is not running${NC}"
fi

# Check 4: Socket File
echo -e "\n${YELLOW}[4] Checking Unix Socket...${NC}"
if [ -S /tmp/atest-ext-ai.sock ]; then
    echo -e "${GREEN}✓ Socket file exists: /tmp/atest-ext-ai.sock${NC}"
    ls -la /tmp/atest-ext-ai.sock
else
    echo -e "${RED}✗ Socket file not found${NC}"
fi

# Check 5: Environment Configuration
echo -e "\n${YELLOW}[5] Setting Environment Variables...${NC}"
export AI_PROVIDER=local
export OLLAMA_ENDPOINT=http://localhost:11434
export AI_MODEL=$CONFIGURED_MODEL
export AI_PLUGIN_SOCKET_PATH=/tmp/atest-ext-ai.sock
export LOG_LEVEL=debug

echo "AI_PROVIDER=$AI_PROVIDER"
echo "OLLAMA_ENDPOINT=$OLLAMA_ENDPOINT"
echo "AI_MODEL=$AI_MODEL"
echo "AI_PLUGIN_SOCKET_PATH=$AI_PLUGIN_SOCKET_PATH"

# Create .env file
echo -e "\n${YELLOW}[6] Creating .env file...${NC}"
cat > .env <<EOF
# AI Configuration
AI_PROVIDER=local
OLLAMA_ENDPOINT=http://localhost:11434
AI_MODEL=$CONFIGURED_MODEL
AI_PLUGIN_SOCKET_PATH=/tmp/atest-ext-ai.sock
LOG_LEVEL=debug

# Plugin Settings
CONFIG_PATH=./configs/default.yaml
EOF
echo -e "${GREEN}✓ .env file created${NC}"

# Test AI functionality
echo -e "\n${YELLOW}[7] Testing AI Functionality...${NC}"
TEST_RESPONSE=$(curl -s -X POST http://localhost:11434/api/generate \
    -d "{\"model\": \"$CONFIGURED_MODEL\", \"prompt\": \"Generate SQL: SELECT all users\", \"stream\": false}" \
    | jq -r '.response' 2>/dev/null | head -5)

if [ ! -z "$TEST_RESPONSE" ]; then
    echo -e "${GREEN}✓ AI Model is responding:${NC}"
    echo "$TEST_RESPONSE"
else
    echo -e "${RED}✗ AI Model is not responding${NC}"
fi

# Recommendation
echo -e "\n==============================================="
echo -e "${YELLOW}RECOMMENDATIONS:${NC}"
echo -e "==============================================="

echo -e "\n${GREEN}1. Restart the AI Plugin with correct configuration:${NC}"
echo "   cd $(pwd)"
echo "   export AI_PROVIDER=local"
echo "   export OLLAMA_ENDPOINT=http://localhost:11434"
echo "   export AI_MODEL=$CONFIGURED_MODEL"
echo "   ./bin/atest-ext-ai"

echo -e "\n${GREEN}2. Or use the development mode:${NC}"
echo "   make dev"

echo -e "\n${GREEN}3. Ensure the main atest configuration includes:${NC}"
echo "   stores:"
echo "     - name: \"ai-assistant\""
echo "       type: \"ai\""
echo "       url: \"unix:///tmp/atest-ext-ai.sock\""

echo -e "\n${GREEN}4. If model issues persist, pull a compatible model:${NC}"
echo "   ollama pull llama2"
echo "   ollama pull codellama"
echo "   ollama pull gemma3:1b"

echo -e "\n${YELLOW}Current Status Summary:${NC}"
if pgrep -f "ollama serve" > /dev/null && [ ! -z "$AVAILABLE_MODELS" ]; then
    echo -e "${GREEN}✓ Ollama is ready with model: $CONFIGURED_MODEL${NC}"
else
    echo -e "${RED}✗ Ollama needs configuration${NC}"
fi

if [ -S /tmp/atest-ext-ai.sock ]; then
    echo -e "${GREEN}✓ AI Plugin socket is available${NC}"
else
    echo -e "${RED}✗ AI Plugin needs to be started${NC}"
fi

echo -e "\n==============================================="
echo "Script completed. Please follow the recommendations above."
echo "==============================================="