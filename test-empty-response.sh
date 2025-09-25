#!/bin/bash

echo "🔍 Testing empty response issue..."

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
echo "📨 Testing potential empty response scenarios..."

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
sleep 5

# 测试2: 需要搜索的问题
echo "📨 Test 2: Search-required question..."
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
        "content": "{\"text\":\"请查询今天北京的天气\"}",
        "mentions": []
      }
    }
  }'

echo ""
echo "⏳ Waiting for processing..."
sleep 5

# 显示应用日志
echo "📋 Application logs:"
echo "===================="
tail -200 app.log

# 停止应用
echo "🛑 Stopping application..."
kill $APP_PID

# 清理
rm -f config.test.yaml feishu_chatgpt app.log

echo "🎉 Empty response test completed!"
echo ""
echo "📋 Look for these debug messages:"
echo "  ✅ Second stage OpenAI call successful"
echo "  📄 Response content length: X"
echo "  📄 Response content: [actual content]"
echo "  ❌ Second stage response is empty (if this appears, it's the problem)"
echo "  📤 Sending response to user..."
echo "  ✅ Response sent successfully"
echo ""
echo "🔍 Common causes of empty responses:"
echo "  - OpenAI API returning empty content"
echo "  - Max tokens too low"
echo "  - Content filtering"
echo "  - Network issues"
echo "  - API rate limiting"
