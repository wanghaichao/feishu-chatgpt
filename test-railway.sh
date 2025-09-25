#!/bin/bash

echo "🧪 Testing Railway deployment locally..."

# 1. 构建镜像
echo "🔨 Building Docker image..."
docker build -f Dockerfile.simple -t feishu-chatgpt-test .

if [ $? -ne 0 ]; then
    echo "❌ Build failed"
    exit 1
fi

echo "✅ Build successful"

# 2. 运行容器
echo "🚀 Starting container..."
docker run --rm -d \
  --name feishu-test \
  -p 8080:8080 \
  -e PORT=8080 \
  -e APP_ID=test \
  -e APP_SECRET=test \
  -e APP_ENCRYPT_KEY=test \
  -e APP_VERIFICATION_TOKEN=test \
  -e OPENAI_KEY=test \
  feishu-chatgpt-test

if [ $? -ne 0 ]; then
    echo "❌ Container start failed"
    exit 1
fi

echo "✅ Container started"

# 3. 等待应用启动
echo "⏳ Waiting for app to start..."
sleep 10

# 4. 测试健康检查
echo "🏥 Testing health check..."
response=$(curl -s http://localhost:8080/ping)

if echo "$response" | grep -q "pong"; then
    echo "✅ Health check passed: $response"
else
    echo "❌ Health check failed: $response"
    docker logs feishu-test
    docker stop feishu-test
    exit 1
fi

# 5. 测试根路径
echo "🏠 Testing root path..."
response=$(curl -s http://localhost:8080/)

if echo "$response" | grep -q "Feishu ChatGPT Bot"; then
    echo "✅ Root path works: $response"
else
    echo "❌ Root path failed: $response"
fi

# 6. 清理
echo "🧹 Cleaning up..."
docker stop feishu-test

echo "🎉 All tests passed! Ready for Railway deployment."
