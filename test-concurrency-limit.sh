#!/bin/bash

echo "🔧 Testing concurrency limit adjustment..."

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
EOF

echo "📋 Test config created"
echo "⚠️  Please update OPENAI_KEY in config.test.yaml with your real OpenAI API key"

# 检查是否有真实的 OpenAI key
if grep -q "sk-proj-your-real-openai-key-here" config.test.yaml; then
    echo "❌ Please update the OpenAI key in config.test.yaml first"
    echo "   Edit the file and replace 'sk-proj-your-real-openai-key-here' with your real OpenAI API key"
    exit 1
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

# 测试需要多个搜索查询的问题
echo "📨 Testing multi-query search with concurrency limit..."
curl -X POST http://localhost:8080/webhook/event \
  -H "Content-Type: application/json" \
  -H "User-Agent: Feishu-Hookshot" \
  -H "X-Request-ID: test_request_concurrency" \
  -d '{
    "schema": "2.0",
    "header": {
      "event_id": "test_event_concurrency",
      "event_type": "im.message.receive_v1",
      "create_time": "1609459200000",
      "token": "test_verification_token",
      "app_id": "cli_test_app_id",
      "tenant_key": "test_tenant"
    },
    "event": {
      "sender": {
        "sender_id": {
          "union_id": "test_user_concurrency",
          "user_id": "test_user_concurrency",
          "open_id": "test_user_concurrency"
        },
        "sender_type": "user",
        "tenant_key": "test_tenant"
      },
      "message": {
        "message_id": "om_test_message_concurrency",
        "root_id": "",
        "parent_id": "",
        "create_time": "1609459200000",
        "chat_id": "oc_test_chat_concurrency",
        "chat_type": "p2p",
        "message_type": "text",
        "content": "{\"text\":\"请查询今天北京的天气情况、最新的科技新闻、人工智能的最新发展、以及最新的股市行情\"}",
        "mentions": []
      }
    }
  }'

echo ""
echo "⏳ Waiting for processing..."
sleep 15

# 显示应用日志
echo "📋 Application logs:"
echo "===================="
tail -100 app.log

# 停止应用
echo "🛑 Stopping application..."
kill $APP_PID

# 清理
rm -f config.test.yaml feishu_chatgpt app.log

echo "🎉 Concurrency limit test completed!"
echo ""
echo "📊 Expected improvements:"
echo "  ✅ Max concurrency: 4 → 3"
echo "  ✅ Reduced timeout failures"
echo "  ✅ Better resource utilization"
echo "  ✅ More stable search results"
echo ""
echo "🔍 Look for these log patterns:"
echo "  🚀 Starting concurrent search for X queries (max concurrency: 3)"
echo "  🔍 [Concurrent] Query X: ..."
echo "  ✅ [Concurrent] Query X successful"
echo "  ❌ [Concurrent] Query X failed (should be fewer now)"
echo "  🎯 [Concurrent] Search completed: X successful, Y failed"
echo ""
echo "💡 Configuration changes applied:"
echo "  - Default max concurrency: 4 → 3"
echo "  - Configurable via SEARCH_MAX_CONCURRENCY environment variable"
echo "  - Hardcoded fallback also updated to 3"
