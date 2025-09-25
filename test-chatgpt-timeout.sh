#!/bin/bash

echo "⏰ Testing ChatGPT timeout configuration..."

# 构建应用
echo "🔨 Building application..."
cd code
go build -o feishu_chatgpt

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

# 测试不同的超时配置
test_timeout() {
    local timeout_sec=$1
    local config_name=$2
    
    echo ""
    echo "🧪 Testing ChatGPT timeout: ${timeout_sec}s ($config_name)"
    
    # 创建测试配置
    cat > config.test.yaml << EOF
APP_ID: "cli_test_app_id"
APP_SECRET: "test_app_secret"
APP_ENCRYPT_KEY: ""
APP_VERIFICATION_TOKEN: "test_verification_token"
BOT_NAME: "test_bot"
OPENAI_KEY: "sk-proj-your-real-openai-key-here"
API_URL: "https://api.openai.com"
HTTP_PORT: 8080
HTTPS_PORT: 8081
USE_HTTPS: false
PROVIDER: "openai"
DEBUG_HTTP: true
SEARCH_ALWAYS: false
SEARCH_TOP_K: 3
SEARCH_OVERALL_TIMEOUT_SEC: 15
SEARCH_PER_FETCH_TIMEOUT_SEC: 10
SEARCH_MAX_CONCURRENCY: 3
CHATGPT_TIMEOUT_SEC: $timeout_sec
EOF

    echo "📋 Test config created"
    echo "⚠️  Please update OPENAI_KEY in config.test.yaml with your real OpenAI API key"

    # 检查是否有真实的 OpenAI key
    if grep -q "sk-proj-your-real-openai-key-here" config.test.yaml; then
        echo "❌ Please update the OpenAI key in config.test.yaml first"
        echo "   Edit the file and replace 'sk-proj-your-real-openai-key-here' with your real OpenAI API key"
        return 1
    fi

    # 设置环境变量
    export PORT=8080

    # 启动应用（后台运行）
    echo "🚀 Starting application..."
    ./feishu_chatgpt -c config.test.yaml > app.log 2>&1 &
    APP_PID=$!

    # 等待应用启动
    echo "⏳ Waiting for app to start..."
    sleep 3

    # 测试健康检查
    echo "🏥 Testing health check..."
    curl -s http://localhost:8080/ping > /dev/null

    # 测试简单问题
    echo "📨 Testing simple question..."
    curl -X POST http://localhost:8080/webhook/event \
      -H "Content-Type: application/json" \
      -H "User-Agent: Feishu-Hookshot" \
      -H "X-Request-ID: test_request_$config_name" \
      -d '{
        "schema": "2.0",
        "header": {
          "event_id": "test_event_1",
          "event_type": "im.message.receive_v1",
          "create_time": "1609459200000",
          "token": "test_verification_token",
          "app_id": "cli_test_app_id",
          "tenant_key": "test_tenant"
        },
        "event": {
          "sender": {
            "sender_id": {
              "union_id": "test_user_1",
              "user_id": "test_user_1",
              "open_id": "test_user_1"
            },
            "sender_type": "user",
            "tenant_key": "test_tenant"
          },
          "message": {
            "message_id": "om_test_message_1",
            "root_id": "",
            "parent_id": "",
            "create_time": "1609459200000",
            "chat_id": "oc_test_chat_1",
            "chat_type": "p2p",
            "message_type": "text",
            "content": "{\"text\":\"你好\"}",
            "mentions": []
          }
        }
      }'

    echo ""
    echo "⏳ Waiting for processing..."
    sleep 8

    # 分析日志
    echo "📊 Analyzing results for $config_name:"
    echo "===================="
    
    # 检查是否有超时相关的日志
    if grep -q "timeout" app.log; then
        echo "⏰ Timeout detected in logs"
        grep "timeout" app.log
    else
        echo "✅ No timeout detected"
    fi
    
    # 检查 ChatGPT 调用日志
    if grep -q "OpenAI Request" app.log; then
        echo "🤖 ChatGPT calls detected:"
        grep "OpenAI Request" app.log
    fi
    
    # 检查响应时间
    if grep -q "Response OK" app.log; then
        echo "📄 Response received:"
        grep "Response OK" app.log
    fi

    # 停止应用
    echo "🛑 Stopping application..."
    kill $APP_PID

    # 清理
    rm -f config.test.yaml app.log
}

# 测试不同的超时设置
test_timeout 10 "short"
test_timeout 30 "default"
test_timeout 60 "long"

echo "🎉 ChatGPT timeout test completed!"
echo ""
echo "📋 Expected behavior:"
echo "  ✅ Short timeout (10s): May timeout on slow responses"
echo "  ✅ Default timeout (30s): Balanced performance"
echo "  ✅ Long timeout (60s): More reliable but slower"
echo ""
echo "🔍 Look for these timeout improvements:"
echo "  - HTTP client timeout: 30s (configurable)"
echo "  - Faster failure detection"
echo "  - Better user experience"
echo "  - Configurable via CHATGPT_TIMEOUT_SEC"
echo ""
echo "💡 Configuration options:"
echo "  - CHATGPT_TIMEOUT_SEC=10  # Fast timeout"
echo "  - CHATGPT_TIMEOUT_SEC=30  # Default timeout"
echo "  - CHATGPT_TIMEOUT_SEC=60  # Long timeout"
