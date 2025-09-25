#!/bin/bash

# Railway 部署优化脚本

echo "🚀 开始 Railway 部署优化..."

# 1. 清理旧的构建缓存
echo "🧹 清理构建缓存..."
docker system prune -f

# 2. 检查 Docker 文件
echo "📋 检查 Docker 文件..."
if [ ! -f "Dockerfile.optimized" ]; then
    echo "❌ Dockerfile.optimized 不存在"
    exit 1
fi

if [ ! -f ".dockerignore" ]; then
    echo "❌ .dockerignore 不存在"
    exit 1
fi

# 3. 本地测试构建
echo "🔨 本地测试构建..."
docker build -f Dockerfile.optimized -t feishu-chatgpt-test .

if [ $? -ne 0 ]; then
    echo "❌ 本地构建失败"
    exit 1
fi

# 4. 检查镜像大小
echo "📊 检查镜像大小..."
docker images feishu-chatgpt-test --format "table {{.Repository}}\t{{.Tag}}\t{{.Size}}"

# 5. 测试运行
echo "🧪 测试运行..."
docker run --rm -d --name feishu-test -p 9000:9000 feishu-chatgpt-test

sleep 5

# 检查容器状态
if docker ps | grep -q feishu-test; then
    echo "✅ 容器运行正常"
    docker stop feishu-test
else
    echo "❌ 容器启动失败"
    docker logs feishu-test
    docker stop feishu-test
    exit 1
fi

echo "🎉 部署准备完成！"
echo "📝 下一步："
echo "   1. 提交代码到 Git"
echo "   2. 在 Railway 中重新部署"
echo "   3. 使用 Dockerfile.optimized 作为构建文件"
