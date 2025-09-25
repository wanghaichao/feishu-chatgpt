# 🔍 详细调试日志说明

本文档说明了为 Feishu ChatGPT Bot 添加的详细调试日志，帮助追踪每个步骤的执行情况和参数。

## 📋 启动阶段日志

### 应用启动流程
```
🚀 Starting Feishu ChatGPT Bot...
📋 Initializing role list...
⚙️ Parsing command line flags...
📁 Config file: ./config.yaml
🔧 Loading configuration...
✅ Config loaded: HTTP_PORT=8080, HTTPS_PORT=8081, USE_HTTPS=false
```

### Railway 端口处理
```
🌐 Using Railway PORT: 8080
✅ Port set to: 8080
```

### 服务初始化
```
🔗 Loading Lark client...
✅ Lark client loaded: APP_ID=cli_xxx, BOT_NAME=chatGpt
🤖 Initializing ChatGPT client...
✅ ChatGPT client initialized: API_URL=https://api.openai.com, PROVIDER=openai
🎯 Initializing handlers...
✅ Handlers initialized
```

### 事件处理器设置
```
📨 Setting up event dispatcher...
✅ Event dispatcher configured: VERIFICATION_TOKEN=test_ver...
🎴 Setting up card action handler...
✅ Card action handler configured
```

### 路由注册
```
🌐 Setting up Gin router...
🛣️ Setting up routes...
  📍 Registering /ping endpoint
  📍 Registering / endpoint
  📍 Registering /health endpoint
  📍 Registering /webhook/event endpoint
  📍 Registering /webhook/card endpoint
✅ All routes registered
```

### 服务器启动
```
🎯 Starting server on port 8080...
🔗 Health check available at: http://localhost:8080/ping
🔗 Webhook endpoint: http://localhost:8080/webhook/event
```

## 📨 消息处理日志

### 消息接收
```
📨 Received message event: om_xxx
🔍 Chat type: singleChat
📝 Message type: text
📋 Message details: msgId=om_xxx, chatId=oc_xxx
🔗 Root ID: om_xxx
🆔 Using msgId as sessionId: om_xxx
📝 Parsed content: 你好，请帮我查询今天的天气
```

### 动作链执行
```
🔄 Starting action chain...
📋 Executing 13 actions in chain
  🔧 Action 1: *handlers.ProcessedUniqueAction
  ✅ Action 1 (*handlers.ProcessedUniqueAction) completed
  🔧 Action 2: *handlers.ProcessMentionAction
  🔍 ProcessMentionAction: handlerType=singleChat
  ✅ Private chat, proceeding
  ✅ Action 2 (*handlers.ProcessMentionAction) completed
  🔧 Action 3: *handlers.AudioAction
  ✅ Action 3 (*handlers.AudioAction) completed
  🔧 Action 4: *handlers.EmptyAction
  🔍 EmptyAction: qParsed='你好，请帮我查询今天的天气' (length=12)
  ✅ Non-empty message, proceeding
  ✅ Action 4 (*handlers.EmptyAction) completed
  ...
```

### 两阶段 ChatGPT 交互

#### 第一阶段：分类
```
🔍 MessageAction: Starting two-stage flow for: '你好，请帮我查询今天的天气'
📋 Session ID: om_xxx
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

#### 第二阶段：Web 搜索（如果需要）
```
🌐 Step 2: Web search required
🔍 Search queries: [今天天气 天气预报]
🔍 Search top K: 3
[Web Search] Query 1: 今天天气
[Web Search] Query 1 context length: 1234
[Web Search] Query 1 context preview: {"title": "今日天气预报", "summary": "..."}
[Web Search] Query 2: 天气预报
[Web Search] Query 2 context length: 987
[Web Search] Query 2 context preview: {"title": "天气预报查询", "summary": "..."}
[Second Stage] built contexts: 2
[Second Stage] Final context JSON length: 2221
[Second Stage] Final context JSON preview: [{"title": "今日天气预报", "summary": "..."}, {"title": "天气预报查询", "summary": "..."}]
🤖 Calling OpenAI for second-stage response...
✅ OpenAI second-stage completed
📄 Second-stage raw: 根据最新的天气预报...
```

## 🏥 健康检查日志

### 请求处理
```
🏥 Health check request from 127.0.0.1
🏠 Root request from 127.0.0.1
💚 Health check request from 127.0.0.1
```

## 🧪 测试方法

### 1. 本地测试启动日志
```bash
./test-detailed-logs.sh
```

### 2. 消息处理流程测试
```bash
./test-message-flow.sh
```

### 3. Railway 部署后查看日志
- 登录 Railway 控制台
- 进入项目页面
- 点击 "Deployments" 标签
- 选择最新的部署
- 点击 "View Logs" 查看详细日志

## 📊 日志级别说明

- 🚀 **启动相关**: 应用启动、配置加载、服务初始化
- 📨 **消息处理**: 消息接收、解析、动作链执行
- 🤖 **AI 交互**: OpenAI API 调用、响应处理
- 🌐 **网络搜索**: Web 搜索查询、结果处理
- 🏥 **健康检查**: 端点访问、状态检查
- ✅ **成功状态**: 操作成功完成
- ❌ **错误状态**: 操作失败或异常
- 🔍 **调试信息**: 详细的参数和状态信息

## 🔧 配置调试

在 `config.yaml` 中设置：
```yaml
DEBUG_HTTP: true  # 启用 HTTP 请求/响应日志
```

## 📝 注意事项

1. **生产环境**: 建议在生产环境中关闭详细日志以提高性能
2. **敏感信息**: 日志中会显示部分配置信息，注意保护敏感数据
3. **日志大小**: 详细日志会增加日志文件大小，注意磁盘空间
4. **性能影响**: 大量日志输出可能影响应用性能

通过这些详细的调试日志，你可以清楚地看到：
- 应用启动的每个步骤
- 消息处理的完整流程
- OpenAI API 的调用和响应
- Web 搜索的执行过程
- 每个动作的执行状态和参数
