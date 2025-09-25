#!/bin/bash

echo "🔍 Testing enhanced debug and fallback mechanisms..."

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

# 测试可能导致空响应的问题
echo "📨 Testing enhanced debug mechanisms..."

# 测试1: 简单问题
echo "📨 Test 1: Simple question..."
curl -X POST http://localhost:8080/webhook/event \
  -H "Content-Type: application/json" \
  -H "User-Agent: Feishu-Hookshot" \
  -H "X-Request-ID: test_request_1" \
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

# 测试2: 可能导致空响应的复杂问题
echo "📨 Test 2: Complex question that might cause empty response..."
curl -X POST http://localhost:8080/webhook/event \
  -H "Content-Type: application/json" \
  -H "User-Agent: Feishu-Hookshot" \
  -H "X-Request-ID: test_request_2" \
  -d '{
    "schema": "2.0",
    "header": {
      "event_id": "test_event_2",
      "event_type": "im.message.receive_v1",
      "create_time": "1609459200000",
      "token": "test_verification_token",
      "app_id": "cli_test_app_id",
      "tenant_key": "test_tenant"
    },
    "event": {
      "sender": {
        "sender_id": {
          "union_id": "test_user_2",
          "user_id": "test_user_2",
          "open_id": "test_user_2"
        },
        "sender_type": "user",
        "tenant_key": "test_tenant"
      },
      "message": {
        "message_id": "om_test_message_2",
        "root_id": "",
        "parent_id": "",
        "create_time": "1609459200000",
        "chat_id": "oc_test_chat_2",
        "chat_type": "p2p",
        "message_type": "text",
        "content": "{\"text\":\"请详细解释量子计算的工作原理、应用领域、技术挑战和未来发展趋势，并分析其对传统计算的影响\"}",
        "mentions": []
      }
    }
  }'

echo ""
echo "⏳ Waiting for processing..."
sleep 8

# 测试3: 需要搜索的问题
echo "📨 Test 3: Search-required question..."
curl -X POST http://localhost:8080/webhook/event \
  -H "Content-Type: application/json" \
  -H "User-Agent: Feishu-Hookshot" \
  -H "X-Request-ID: test_request_3" \
  -d '{
    "schema": "2.0",
    "header": {
      "event_id": "test_event_3",
      "event_type": "im.message.receive_v1",
      "create_time": "1609459200000",
      "token": "test_verification_token",
      "app_id": "cli_test_app_id",
      "tenant_key": "test_tenant"
    },
    "event": {
      "sender": {
        "sender_id": {
          "union_id": "test_user_3",
          "user_id": "test_user_3",
          "open_id": "test_user_3"
        },
        "sender_type": "user",
        "tenant_key": "test_tenant"
      },
      "message": {
        "message_id": "om_test_message_3",
        "root_id": "",
        "parent_id": "",
        "create_time": "1609459200000",
        "chat_id": "oc_test_chat_3",
        "chat_type": "p2p",
        "message_type": "text",
        "content": "{\"text\":\"请查询今天北京的天气情况\"}",
        "mentions": []
      }
    }
  }'

echo ""
echo "⏳ Waiting for processing..."
sleep 8

# 显示应用日志
echo "📋 Application logs:"
echo "===================="
tail -300 app.log

# 停止应用
echo "🛑 Stopping application..."
kill $APP_PID

# 清理
rm -f config.test.yaml feishu_chatgpt app.log

echo "🎉 Enhanced debug test completed!"
echo ""
echo "📋 Look for these enhanced debug messages:"
echo "  [OpenAI Request] Model: gpt-5-2025-08-07, MaxTokens: X, Messages: Y"
echo "  [OpenAI Response] Error: <nil>, Choices count: X"
echo "  [OpenAI Response] Full response: {...}"
echo "  [OpenAI Response] First choice role: assistant"
echo "  [OpenAI Response] First choice content length: X"
echo "  [OpenAI Response] First choice content: '...'"
echo "  [OpenAI Response] Finish reason: stop (if available)"
echo ""
echo "🔍 Enhanced fallback mechanisms:"
echo "  ⚠️ Second stage response is empty, retrying with higher max_tokens..."
echo "  🔄 Retrying with max_tokens: X"
echo "  ❌ Retry also returned empty response, trying fallback approach..."
echo "  🔄 Trying simple approach with max_tokens: 2000"
echo "  ✅ Simple approach successful, got response: ..."
echo ""
echo "🎯 Expected improvements:"
echo "  - Detailed API response analysis"
echo "  - Multi-level fallback mechanisms"
echo "  - Better error diagnosis"
echo "  - Higher success rate for edge cases"
