#!/bin/bash

echo "🚀 Testing concurrent search performance..."

# 构建应用
echo "🔨 Building application..."
cd code
go build -o feishu_chatgpt

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

# 创建测试配置（启用并发搜索）
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
# 并发搜索配置
SEARCH_MAX_CONCURRENCY: 6
SEARCH_OVERALL_TIMEOUT_SEC: 15
SEARCH_PER_FETCH_TIMEOUT_SEC: 8
EOF

echo "📋 Test config created with concurrent search settings"
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

# 记录开始时间
start_time=$(date +%s)

# 模拟发送一个需要多查询搜索的消息
echo "📨 Simulating webhook message with multiple search queries..."
curl -X POST http://localhost:8080/webhook/event \
  -H "Content-Type: application/json" \
  -H "User-Agent: Feishu-Hookshot" \
  -H "X-Request-ID: test_request_123" \
  -d '{
    "schema": "2.0",
    "header": {
      "event_id": "test_event_123",
      "event_type": "im.message.receive_v1",
      "create_time": "1609459200000",
      "token": "test_verification_token",
      "app_id": "cli_test_app_id",
      "tenant_key": "test_tenant"
    },
    "event": {
      "sender": {
        "sender_id": {
          "union_id": "test_user_123",
          "user_id": "test_user_123",
          "open_id": "test_user_123"
        },
        "sender_type": "user",
        "tenant_key": "test_tenant"
      },
      "message": {
        "message_id": "om_test_message_123",
        "root_id": "",
        "parent_id": "",
        "create_time": "1609459200000",
        "chat_id": "oc_test_chat_123",
        "chat_type": "p2p",
        "message_type": "text",
        "content": "{\"text\":\"请帮我查询今天的天气、最新的科技新闻、股票市场动态和体育赛事结果\"}",
        "mentions": []
      }
    }
  }'

echo ""
echo "⏳ Waiting for concurrent search processing..."
sleep 10

# 记录结束时间
end_time=$(date +%s)
duration=$((end_time - start_time))

echo "⏱️ Total processing time: ${duration} seconds"

# 显示应用日志
echo "📋 Application logs:"
echo "===================="
tail -100 app.log

# 停止应用
echo "🛑 Stopping application..."
kill $APP_PID

# 清理
rm -f config.test.yaml feishu_chatgpt app.log

echo "🎉 Concurrent search test completed!"
echo ""
echo "📊 Performance Summary:"
echo "  ⏱️ Total time: ${duration} seconds"
echo "  🚀 Concurrent searches: Up to 6 simultaneous"
echo "  ⏰ Per-query timeout: 8 seconds"
echo "  ⏰ Overall timeout: 15 seconds"
echo ""
echo "📋 Expected logs should include:"
echo "  🚀 Starting concurrent search for X queries (max concurrency: 6)..."
echo "  🔍 [Concurrent] Query 1: 今天天气 (topK=3)"
echo "  🔍 [Concurrent] Query 2: 科技新闻 (topK=3)"
echo "  🔍 [Concurrent] Query 3: 股票市场 (topK=3)"
echo "  🔍 [Concurrent] Query 4: 体育赛事 (topK=3)"
echo "  ✅ [Concurrent] Query X context length: XXXX chars"
echo "  🎯 [Concurrent] Search completed: X successful, X failed"
echo ""
echo "💡 Performance benefits:"
echo "  - Multiple queries run simultaneously instead of sequentially"
echo "  - Reduced total waiting time"
echo "  - Better resource utilization"
echo "  - Timeout protection prevents hanging"
