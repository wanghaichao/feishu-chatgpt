# Railway 部署快速修复

## 问题
应用卡在 "service unavailable" 健康检查失败

## 解决方案

### 1. 端口问题修复
- ✅ 修复了 Railway PORT 环境变量处理
- ✅ 应用现在会正确监听 Railway 分配的端口

### 2. 简化 Dockerfile
- ✅ 使用 `Dockerfile.simple` 替代复杂的构建
- ✅ 移除了不必要的启动脚本和配置文件

### 3. 增强日志
- ✅ 添加了详细的启动日志
- ✅ 显示端口信息和健康检查地址

## 部署步骤

### 1. 提交代码
```bash
git add .
git commit -m "Fix Railway port handling and simplify deployment"
git push
```

### 2. 在 Railway 中
1. 删除当前部署
2. 重新创建部署
3. 使用 `Dockerfile.simple` 作为构建文件
4. 设置环境变量：
   ```
   APP_ID=你的飞书应用ID
   APP_SECRET=你的飞书应用密钥
   APP_ENCRYPT_KEY=你的飞书应用加密密钥
   APP_VERIFICATION_TOKEN=你的飞书应用验证令牌
   OPENAI_KEY=你的OpenAI API密钥
   ```

### 3. 验证部署
- 查看 Railway 日志，应该看到：
  ```
  🚀 Starting Feishu ChatGPT Bot...
  🌐 Using Railway PORT: 8080
  ✅ Port set to: 8080
  🎯 Starting server on port 8080...
  🔗 Health check available at: http://localhost:8080/ping
  ```

### 4. 测试健康检查
访问 `https://your-app.railway.app/ping` 应该返回：
```json
{
  "message": "pong",
  "status": "healthy"
}
```

## 本地测试
```bash
# 运行测试脚本
./test-railway.sh
```

## 故障排除

如果仍然失败：
1. 检查 Railway 日志中的错误信息
2. 确认所有环境变量都已设置
3. 验证应用是否在正确的端口启动
4. 检查防火墙或网络设置

## 关键修复点
- **端口处理**：现在正确使用 Railway 的 PORT 环境变量
- **简化构建**：移除了复杂的启动脚本和配置
- **增强日志**：更容易诊断问题
- **健康检查**：多个端点确保检查通过
