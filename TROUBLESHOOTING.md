# 🔧 问题排查指南

## 📋 问题描述
用户报告："发起消息，没有任何日志，也没有返回，没有去查询chatgpt和websearch"

## 🔍 排查过程

### 1. 添加详细调试日志
我们为应用添加了全面的调试日志，包括：
- 应用启动流程
- 消息接收和处理
- 动作链执行
- OpenAI API 调用
- Web 搜索过程

### 2. 测试结果分析

#### ✅ 成功的部分：
1. **应用启动正常**：所有服务都正确初始化
2. **Webhook 请求到达**：请求成功到达 `/webhook/event` 端点
3. **消息处理链启动**：`MessageAction` 被正确调用
4. **调试日志工作**：可以看到完整的执行流程

#### ❌ 问题所在：
1. **OpenAI API 调用失败**：`❌ OpenAI classification failed: openai 请求失败`
2. **动作链提前终止**：`⏹️ Action 13 (*handlers.MessageAction) returned false, stopping chain`

## 🎯 根本原因

### 问题 1：OpenAI API Key 无效
```
❌ OpenAI classification failed: openai 请求失败
```

**解决方案**：
1. 检查 `config.yaml` 中的 `OPENAI_KEY` 是否正确
2. 确保 OpenAI API Key 有足够的余额
3. 验证 API Key 的权限设置

### 问题 2：飞书配置问题
```
[Warn] [custom app tenantAccessToken cache, err:{
  CodeError: {
    Code: 10003,
    Msg: "invalid param"
  }
}]
```

**解决方案**：
1. 检查 `APP_ID` 和 `APP_SECRET` 是否正确
2. 确保飞书应用配置正确
3. 验证 webhook URL 设置

## 🛠️ 解决步骤

### 步骤 1：验证 OpenAI 配置
```bash
# 检查配置文件
cat code/config.yaml | grep OPENAI_KEY

# 测试 OpenAI API
curl -H "Authorization: Bearer YOUR_OPENAI_KEY" \
     -H "Content-Type: application/json" \
     -d '{"model": "gpt-3.5-turbo", "messages": [{"role": "user", "content": "Hello"}]}' \
     https://api.openai.com/v1/chat/completions
```

### 步骤 2：验证飞书配置
1. 登录飞书开放平台
2. 检查应用配置：
   - APP_ID
   - APP_SECRET
   - APP_ENCRYPT_KEY
   - APP_VERIFICATION_TOKEN
3. 验证 webhook URL 设置

### 步骤 3：测试消息处理
```bash
# 使用真实配置测试
./test-with-mock-openai.sh
```

## 📊 调试日志说明

### 启动阶段日志
```
🚀 Starting Feishu ChatGPT Bot...
📋 Initializing role list...
⚙️ Parsing command line flags...
🔧 Loading configuration...
✅ Config loaded: HTTP_PORT=8080, HTTPS_PORT=8081, USE_HTTPS=false
🌐 Using Railway PORT: 8080
✅ Port set to: 8080
🔗 Loading Lark client...
✅ Lark client loaded: APP_ID=cli_xxx, BOT_NAME=chatGpt
🤖 Initializing ChatGPT client...
✅ ChatGPT client initialized: API_URL=https://api.openai.com, PROVIDER=openai
```

### 消息处理日志
```
📨 Webhook event received from ::1
📋 Request headers: map[Content-Type:[application/json] ...]
📝 Request body length: 901
🎯 Handler called with event: om_test_message_123
📋 Event details: chatType=p2p, msgType=text
📨 Received message event: om_test_message_123
🔍 Chat type: singleChat
📝 Message type: text
📋 Message details: msgId=om_test_message_123, chatId=oc_test_chat_123
🆔 Using msgId as sessionId: om_test_message_123
📝 Parsed content: 你好，请帮我查询今天的天气
🔄 Starting action chain...
📋 Executing 13 actions in chain
```

### 两阶段 ChatGPT 交互日志
```
🔍 MessageAction: Starting two-stage flow for: '你好，请帮我查询今天的天气'
📋 Session ID: om_test_message_123
🎯 Step 1: Building classification prompt...
📚 Getting session history...
📖 Session history length: 0 messages
🔧 Building classification messages...
📝 Total messages to send: 2
🤖 Calling OpenAI for classification...
✅ OpenAI classification completed
📄 Raw response: {"need_web": true, "queries": ["今天天气", "天气预报"], "search_top_k": 3}
🔍 Parsing classification result...
✅ Classification parsed successfully
📊 Decision: {"need_web":true,"queries":["今天天气","天气预报"],"search_top_k":3}
🔍 Decision details: need_web=true, queries_count=2, search_top_k=3
```

## 🧪 测试方法

### 1. 基础功能测试
```bash
./test-detailed-logs.sh
```

### 2. Webhook 测试
```bash
./test-webhook.sh
```

### 3. 完整流程测试
```bash
./test-with-mock-openai.sh
```

## 📝 配置检查清单

### OpenAI 配置
- [ ] `OPENAI_KEY` 有效且有余额
- [ ] `API_URL` 正确（默认：https://api.openai.com）
- [ ] `PROVIDER` 设置为 "openai"

### 飞书配置
- [ ] `APP_ID` 正确
- [ ] `APP_SECRET` 正确
- [ ] `APP_ENCRYPT_KEY` 正确（如果使用加密）
- [ ] `APP_VERIFICATION_TOKEN` 正确
- [ ] `BOT_NAME` 设置正确

### 服务器配置
- [ ] `HTTP_PORT` 设置正确
- [ ] Railway `PORT` 环境变量正确映射
- [ ] Webhook URL 在飞书平台正确配置

## 🚀 部署到 Railway

1. **设置环境变量**：
   ```
   APP_ID=your_app_id
   APP_SECRET=your_app_secret
   APP_ENCRYPT_KEY=your_encrypt_key
   APP_VERIFICATION_TOKEN=your_verification_token
   OPENAI_KEY=your_openai_key
   ```

2. **查看日志**：
   - 在 Railway 控制台点击 "View Logs"
   - 应该能看到详细的启动和请求处理日志

3. **验证部署**：
   - 访问 `https://your-app.railway.app/ping`
   - 应该返回 `{"message": "pong", "status": "healthy"}`

## 📞 进一步支持

如果问题仍然存在，请提供：
1. 完整的应用日志
2. 配置文件（隐藏敏感信息）
3. Railway 部署日志
4. 具体的错误信息

通过这些详细的调试日志，我们可以准确定位问题出现在哪个环节，并快速解决。
