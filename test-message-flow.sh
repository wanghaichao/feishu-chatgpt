#!/bin/bash

echo "🧪 Testing message processing flow with detailed logs..."

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
OPENAI_KEY: "sk-test-key-123456789"
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
./feishu_chatgpt -c config.test.yaml > app.log 2>&1 &
APP_PID=$!

# 等待应用启动
echo "⏳ Waiting for app to start..."
sleep 3

# 测试健康检查
echo "🏥 Testing health check..."
curl -s http://localhost:8080/ping > /dev/null

# 模拟发送一个 webhook 消息
echo "📨 Simulating webhook message..."
curl -X POST http://localhost:8080/webhook/event \
  -H "Content-Type: application/json" \
  -d '{
    "schema": "2.0",
    "header": {
      "event_id": "test_event_123",
      "event_type": "im.message.receive_v1",
      "create_time": "1609459200000",
      "token": "test_verification_token",
      "app_id": "test_app_id",
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
        "content": "{\"text\":\"你好，请帮我查询今天的天气\"}",
        "mentions": []
      }
    }
  }' > /dev/null

echo "⏳ Waiting for message processing..."
sleep 2

# 显示应用日志
echo "📋 Application logs:"
echo "===================="
tail -50 app.log

# 停止应用
echo "🛑 Stopping application..."
kill $APP_PID

# 清理
rm -f config.test.yaml feishu_chatgpt app.log

echo "🎉 Message flow test completed!"
echo ""
echo "📋 Expected message processing logs should include:"
echo "  📨 Received message event: om_test_message_123"
echo "  🔍 Chat type: singleChat"
echo "  📝 Message type: text"
echo "  📋 Message details: msgId=om_test_message_123, chatId=oc_test_chat_123"
echo "  🆔 Using msgId as sessionId: om_test_message_123"
echo "  📝 Parsed content: 你好，请帮我查询今天的天气"
echo "  🔄 Starting action chain..."
echo "  📋 Executing 13 actions in chain"
echo "  🔧 Action 1: *handlers.ProcessedUniqueAction"
echo "  ✅ Action 1 (*handlers.ProcessedUniqueAction) completed"
echo "  🔧 Action 2: *handlers.ProcessMentionAction"
echo "  🔍 ProcessMentionAction: handlerType=singleChat"
echo "  ✅ Private chat, proceeding"
echo "  ✅ Action 2 (*handlers.ProcessMentionAction) completed"
echo "  ..."
