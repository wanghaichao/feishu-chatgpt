# Railway 部署指南

## 问题解决

### 健康检查失败问题
应用卡在 "Performing healthchecks" 通常是因为：
1. 应用没有正确监听 Railway 分配的端口
2. 健康检查端点不可用
3. 应用启动失败

## 解决方案

### 1. 使用优化的 Dockerfile
```bash
# 使用 Dockerfile.railway 而不是默认的 Dockerfile
```

### 2. 设置环境变量
在 Railway 项目设置中添加以下环境变量：

**必需的环境变量：**
- `APP_ID`: 你的飞书应用 ID
- `APP_SECRET`: 你的飞书应用密钥
- `APP_ENCRYPT_KEY`: 你的飞书应用加密密钥
- `APP_VERIFICATION_TOKEN`: 你的飞书应用验证令牌
- `OPENAI_KEY`: 你的 OpenAI API 密钥

**可选的环境变量：**
- `BOT_NAME`: 机器人名称（默认：chatGpt）
- `API_URL`: OpenAI API 地址（默认：https://api.openai.com）
- `GOOGLE_API_KEY`: Google 搜索 API 密钥（可选）
- `GOOGLE_CSE_ID`: Google 自定义搜索引擎 ID（可选）

### 3. 健康检查配置
Railway 会自动检查以下端点：
- `GET /ping` - 返回 `{"message": "pong", "status": "healthy"}`
- `GET /` - 返回 `{"message": "Feishu ChatGPT Bot is running", "status": "healthy"}`
- `GET /health` - 返回 `{"status": "healthy", "service": "feishu-chatgpt"}`

### 4. 部署步骤

1. **提交代码**：
   ```bash
   git add .
   git commit -m "Fix Railway deployment health checks"
   git push
   ```

2. **在 Railway 中**：
   - 删除当前部署
   - 重新创建部署
   - 选择 "Dockerfile.railway" 作为构建文件
   - 设置所有必需的环境变量

3. **验证部署**：
   - 等待构建完成
   - 检查健康检查是否通过
   - 访问 `https://your-app.railway.app/ping` 验证应用运行

### 5. 故障排除

**如果仍然卡在健康检查**：
1. 检查 Railway 日志中的错误信息
2. 确认所有环境变量都已正确设置
3. 验证应用是否在正确的端口上启动
4. 检查防火墙设置

**常见错误**：
- `APP_ID not set`: 飞书应用 ID 未设置
- `APP_SECRET not set`: 飞书应用密钥未设置
- `OPENAI_KEY not set`: OpenAI API 密钥未设置
- `Port already in use`: 端口冲突（Railway 会自动处理）

### 6. 测试本地构建

```bash
# 运行部署脚本
./deploy.sh

# 或者手动测试
docker build -f Dockerfile.railway -t feishu-chatgpt-test .
docker run --rm -d --name feishu-test -p 9000:9000 \
  -e APP_ID=your_app_id \
  -e APP_SECRET=your_app_secret \
  -e OPENAI_KEY=your_openai_key \
  feishu-chatgpt-test
```

## 配置示例

### Railway 环境变量设置
```
APP_ID=cli_axxx
APP_SECRET=xxx
APP_ENCRYPT_KEY=xxx
APP_VERIFICATION_TOKEN=xxx
OPENAI_KEY=sk-xxx
BOT_NAME=chatGpt
API_URL=https://api.openai.com
```

### 飞书应用配置
确保你的飞书应用配置了正确的：
- 事件订阅 URL: `https://your-app.railway.app/webhook/event`
- 卡片回调 URL: `https://your-app.railway.app/webhook/card`
- 权限范围：消息接收、卡片交互等
