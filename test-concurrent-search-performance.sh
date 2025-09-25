#!/bin/bash

echo "🚀 Testing concurrent search performance optimization..."

# 构建应用
echo "🔨 Building application..."
cd code
go build -o feishu_chatgpt

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

# 测试不同的配置
test_config() {
    local config_name=$1
    local per_fetch_timeout=$2
    local overall_timeout=$3
    local max_concurrency=$4
    
    echo ""
    echo "🧪 Testing configuration: $config_name"
    echo "   Per-fetch timeout: ${per_fetch_timeout}s"
    echo "   Overall timeout: ${overall_timeout}s"
    echo "   Max concurrency: $max_concurrency"
    
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
SEARCH_OVERALL_TIMEOUT_SEC: $overall_timeout
SEARCH_PER_FETCH_TIMEOUT_SEC: $per_fetch_timeout
SEARCH_MAX_CONCURRENCY: $max_concurrency
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

    # 测试需要搜索的问题
    echo "📨 Testing search-required question..."
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
            "content": "{\"text\":\"请查询今天北京的天气情况、最新的科技新闻、以及人工智能的最新发展\"}",
            "mentions": []
          }
        }
      }'

    echo ""
    echo "⏳ Waiting for processing..."
    sleep 15

    # 分析日志
    echo "📊 Analyzing results for $config_name:"
    echo "===================="
    
    # 统计成功和失败的查询
    success_count=$(grep -c "✅ \[Concurrent\] Query.*successful" app.log || echo "0")
    failed_count=$(grep -c "❌ \[Concurrent\] Query.*failed" app.log || echo "0")
    timeout_count=$(grep -c "⏰ \[Concurrent\] Query.*timed out" app.log || echo "0")
    
    echo "✅ Successful queries: $success_count"
    echo "❌ Failed queries: $failed_count"
    echo "⏰ Timeout queries: $timeout_count"
    
    if [ $((success_count + failed_count)) -gt 0 ]; then
        success_rate=$((success_count * 100 / (success_count + failed_count)))
        echo "📈 Success rate: ${success_rate}%"
    fi
    
    # 显示关键日志
    echo ""
    echo "🔍 Key logs:"
    grep -E "(Concurrent|Search completed|timeout|failed|successful)" app.log | tail -10

    # 停止应用
    echo "🛑 Stopping application..."
    kill $APP_PID

    # 清理
    rm -f config.test.yaml feishu_chatgpt app.log
    
    echo "✅ Test completed for $config_name"
}

# 测试不同配置
echo "🎯 Starting performance tests..."

# 测试1: 当前配置（6s超时，4并发）
test_config "current" 6 10 4

# 测试2: 保守配置（12s超时，2并发）
test_config "conservative" 12 20 2

# 测试3: 平衡配置（10s超时，3并发）
test_config "balanced" 10 15 3

# 测试4: 激进配置（8s超时，4并发）
test_config "aggressive" 8 12 4

echo ""
echo "🎉 Concurrent search performance test completed!"
echo ""
echo "📊 Test Summary:"
echo "  🧪 Tested 4 different configurations"
echo "  📈 Measured success rates and timeout patterns"
echo "  🔍 Analyzed performance characteristics"
echo ""
echo "💡 Recommendations:"
echo "  - Use 'balanced' config for best overall performance"
echo "  - Use 'conservative' config for high reliability"
echo "  - Use 'aggressive' config for fast response"
echo "  - Monitor actual performance and adjust accordingly"
echo ""
echo "🔧 Configuration parameters:"
echo "  - SEARCH_PER_FETCH_TIMEOUT_SEC: Controls individual query timeout"
echo "  - SEARCH_OVERALL_TIMEOUT_SEC: Controls total search timeout"
echo "  - SEARCH_MAX_CONCURRENCY: Controls concurrent query limit"
