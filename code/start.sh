#!/bin/bash

# Railway 启动脚本

echo "🚀 Starting Feishu ChatGPT Bot..."

# 检查必要的环境变量
if [ -z "$APP_ID" ]; then
    echo "❌ APP_ID 环境变量未设置"
    exit 1
fi

if [ -z "$APP_SECRET" ]; then
    echo "❌ APP_SECRET 环境变量未设置"
    exit 1
fi

if [ -z "$OPENAI_KEY" ]; then
    echo "❌ OPENAI_KEY 环境变量未设置"
    exit 1
fi

# 检查端口
if [ -z "$PORT" ]; then
    echo "⚠️  PORT 环境变量未设置，使用默认端口 9000"
    export PORT=9000
fi

echo "✅ 环境变量检查通过"
echo "📡 监听端口: $PORT"
echo "🔧 使用配置文件: config.railway.yaml"

# 启动应用
exec /app/feishu_chatgpt -c config.railway.yaml
