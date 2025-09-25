# ⏰ ChatGPT 超时优化

## 📋 问题分析

用户反馈：
```
websearch要有超时，chatgpt没有超时，等待
```

**问题分析**：
- Web search 有超时控制（6-20秒）
- ChatGPT API 调用没有超时控制（默认110秒）
- 导致整个流程在等待 ChatGPT 响应时卡住
- 用户体验差，系统响应慢

## 🛠️ 优化措施

### 1. 添加 ChatGPT 超时配置

**修改文件**：`code/initialization/config.go`

**新增配置字段**：
```go
type Config struct {
    // ... 其他配置 ...
    // ChatGPT API timeout in seconds
    ChatGPTTimeoutSec int
}
```

**配置加载**：
```go
ChatGPTTimeoutSec: getViperIntValue("CHATGPT_TIMEOUT_SEC", 30),
```

### 2. 修改 HTTP 客户端超时设置

**修改文件**：`code/services/openai/common.go`

**修改前**：
```go
func (gpt ChatGPT) sendRequestWithBodyType(link, method string, bodyType requestBodyType,
    requestBody interface{}, responseBody interface{}) error {
    var err error
    client := &http.Client{Timeout: 110 * time.Second}  // 固定110秒
    // ...
}
```

**修改后**：
```go
func (gpt ChatGPT) sendRequestWithBodyType(link, method string, bodyType requestBodyType,
    requestBody interface{}, responseBody interface{}) error {
    var err error
    
    // 使用配置的超时时间，默认30秒
    timeout := 30 * time.Second
    if gpt.ChatGPTTimeoutSec > 0 {
        timeout = time.Duration(gpt.ChatGPTTimeoutSec) * time.Second
    }
    
    client := &http.Client{Timeout: timeout}
    // ...
}
```

### 3. 更新 ChatGPT 结构体

**新增字段**：
```go
type ChatGPT struct {
    // ... 其他字段 ...
    // ChatGPT API timeout in seconds
    ChatGPTTimeoutSec int
}
```

**构造函数更新**：
```go
func NewChatGPT(config initialization.Config) *ChatGPT {
    // ...
    return &ChatGPT{
        // ... 其他字段 ...
        ChatGPTTimeoutSec: config.ChatGPTTimeoutSec,
    }
}
```

## 🎯 优化效果

### 优化前
```
⏰ [Concurrent] Query 1 timed out after 6s
❌ [Concurrent] Query 1 failed: search timeout after 6s
🎯 [Concurrent] Search completed: 2 successful, 1 failed
✅ [Second Stage] Using 2 successful search results, ignoring 1 failed searches
📤 Sending response to user...
⏳ Waiting for ChatGPT response... (可能等待110秒)
```

**问题**：ChatGPT 调用没有超时，可能等待很长时间

### 优化后
```
⏰ [Concurrent] Query 1 timed out after 6s
❌ [Concurrent] Query 1 failed: search timeout after 6s
🎯 [Concurrent] Search completed: 2 successful, 1 failed
✅ [Second Stage] Using 2 successful search results, ignoring 1 failed searches
📤 Sending response to user...
⏰ ChatGPT timeout after 30s (可配置)
```

**改进**：ChatGPT 调用有超时控制，快速失败

## 📊 超时配置对比

| 组件 | 优化前 | 优化后 | 改善 |
|------|--------|--------|------|
| **Web Search** | ✅ 6-20秒超时 | ✅ 6-20秒超时 | 无变化 |
| **ChatGPT API** | ❌ 110秒固定超时 | ✅ 30秒可配置超时 | 显著改善 |
| **整体响应** | ⚠️ 可能等待110秒 | ✅ 最多等待30秒 | 显著改善 |
| **用户体验** | ❌ 响应慢 | ✅ 响应快 | 显著改善 |

## 🔧 配置选项

### 环境变量配置
```bash
# 快速超时（适合快速响应）
export CHATGPT_TIMEOUT_SEC=10

# 默认超时（平衡性能）
export CHATGPT_TIMEOUT_SEC=30

# 长超时（适合复杂查询）
export CHATGPT_TIMEOUT_SEC=60
```

### 配置文件设置
```yaml
# config.yaml
CHATGPT_TIMEOUT_SEC: 30
```

## 🧪 测试验证

### 使用测试脚本
```bash
./test-chatgpt-timeout.sh
```

### 测试场景
1. **短超时测试**：10秒超时，测试快速失败
2. **默认超时测试**：30秒超时，测试平衡性能
3. **长超时测试**：60秒超时，测试复杂查询

### 预期行为
```
🧪 Testing ChatGPT timeout: 10s (short)
⏰ Timeout detected in logs
✅ No timeout detected
🤖 ChatGPT calls detected:
📄 Response received:
```

## 🎉 主要优势

### 1. 提高响应速度
- **快速失败**：ChatGPT 调用超时后快速失败
- **用户友好**：用户不会等待过长时间
- **系统稳定**：避免长时间阻塞

### 2. 可配置性
- **灵活配置**：可以根据需要调整超时时间
- **环境适配**：不同环境可以设置不同的超时值
- **性能调优**：可以根据网络情况优化

### 3. 一致性
- **统一超时**：Web search 和 ChatGPT 都有超时控制
- **可预测性**：系统行为更加可预测
- **可靠性**：减少系统卡死的可能性

### 4. 用户体验
- **快速响应**：用户不会等待过长时间
- **明确反馈**：超时后会有明确的错误信息
- **可重试**：用户可以重新发送请求

## 🚀 部署建议

### 1. 立即生效
- 重启应用即可生效
- 无需额外配置

### 2. 推荐配置
```bash
# 生产环境推荐
export CHATGPT_TIMEOUT_SEC=30

# 开发环境推荐
export CHATGPT_TIMEOUT_SEC=60

# 测试环境推荐
export CHATGPT_TIMEOUT_SEC=10
```

### 3. 监控指标
- ChatGPT 调用成功率
- 平均响应时间
- 超时频率
- 用户满意度

## 🎯 总结

通过这次优化，我们实现了：

✅ **ChatGPT 超时控制**：从110秒固定超时改为30秒可配置超时
✅ **快速失败机制**：超时后快速失败，不阻塞系统
✅ **可配置性**：通过环境变量灵活配置超时时间
✅ **一致性**：Web search 和 ChatGPT 都有超时控制
✅ **用户体验**：显著提高响应速度和系统稳定性

这个改进解决了 ChatGPT 调用没有超时的问题，确保系统能够快速响应，避免长时间等待，大大提升了用户体验！
