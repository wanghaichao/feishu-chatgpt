#!/bin/bash

echo "🧪 Testing detailed logging with all debug information..."

# 构建应用
echo "🔨 Building application..."
cd code
go build -o feishu_chatgpt

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

# 创建测试配置
cat > config.test.yaml << EOF
APP_ID: "test_app_id"
APP_SECRET: "test_app_secret"
APP_ENCRYPT_KEY: "test_encrypt_key"
APP_VERIFICATION_TOKEN: "test_verification_token"
BOT_NAME: "test_bot"
OPENAI_KEY: "test_openai_key"
API_URL: "https://api.openai.com"
HTTP_PORT: 8080
HTTPS_PORT: 8081
USE_HTTPS: false
PROVIDER: "openai"
DEBUG_HTTP: true
SEARCH_ALWAYS: false
SEARCH_TOP_K: 3
EOF

echo "📋 Test config created"

# 设置环境变量
export PORT=8080

# 启动应用（后台运行）
echo "🚀 Starting application..."
./feishu_chatgpt -c config.test.yaml &
APP_PID=$!

# 等待应用启动
echo "⏳ Waiting for app to start..."
sleep 3

# 测试健康检查
echo "🏥 Testing health check..."
curl -s http://localhost:8080/ping
echo ""

echo "🏠 Testing root endpoint..."
curl -s http://localhost:8080/
echo ""

echo "💚 Testing health endpoint..."
curl -s http://localhost:8080/health
echo ""

# 停止应用
echo "🛑 Stopping application..."
kill $APP_PID

# 清理
rm -f config.test.yaml feishu_chatgpt

echo "🎉 Detailed logging test completed!"
echo ""
echo "📋 Expected log output should include:"
echo "  🚀 Starting Feishu ChatGPT Bot..."
echo "  📋 Initializing role list..."
echo "  ⚙️ Parsing command line flags..."
echo "  🔧 Loading configuration..."
echo "  🌐 Using Railway PORT: 8080"
echo "  🔗 Loading Lark client..."
echo "  🤖 Initializing ChatGPT client..."
echo "  🎯 Initializing handlers..."
echo "  📨 Setting up event dispatcher..."
echo "  🎴 Setting up card action handler..."
echo "  🌐 Setting up Gin router..."
echo "  🛣️ Setting up routes..."
echo "  📍 Registering /ping endpoint"
echo "  📍 Registering / endpoint"
echo "  📍 Registering /health endpoint"
echo "  📍 Registering /webhook/event endpoint"
echo "  📍 Registering /webhook/card endpoint"
echo "  ✅ All routes registered"
echo "  🎯 Starting server on port 8080..."
echo "  🔗 Health check available at: http://localhost:8080/ping"
echo "  🔗 Webhook endpoint: http://localhost:8080/webhook/event"
