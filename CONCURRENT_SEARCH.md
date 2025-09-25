# 🚀 并发搜索优化

## 📋 优化概述

为了提高 web 搜索的响应速度，我们实现了并发搜索功能，可以同时执行多个搜索查询，显著减少等待时间。

## 🔧 技术实现

### 1. 并发控制
- **信号量机制**：使用 `sync.WaitGroup` 和 channel 信号量控制并发数
- **可配置并发数**：通过 `SEARCH_MAX_CONCURRENCY` 配置最大并发数（默认4，最大10）
- **资源保护**：避免同时启动过多 goroutine 导致资源耗尽

### 2. 超时控制
- **单查询超时**：`SEARCH_PER_FETCH_TIMEOUT_SEC` 控制单个搜索的超时时间（默认6秒）
- **整体超时**：`SEARCH_OVERALL_TIMEOUT_SEC` 控制整个搜索过程的最大时间（默认10秒）
- **优雅降级**：超时后继续处理已完成的搜索结果

### 3. 错误处理
- **失败重试**：Google 搜索失败时自动回退到 DuckDuckGo
- **部分成功**：即使部分查询失败，仍使用成功的查询结果
- **详细日志**：记录每个查询的成功/失败状态

## ⚙️ 配置参数

在 `config.yaml` 中添加以下配置：

```yaml
# 并发搜索配置
SEARCH_MAX_CONCURRENCY: 6        # 最大并发数 (1-10)
SEARCH_OVERALL_TIMEOUT_SEC: 15    # 整体超时时间（秒）
SEARCH_PER_FETCH_TIMEOUT_SEC: 8  # 单查询超时时间（秒）
```

### 配置说明

| 参数 | 默认值 | 范围 | 说明 |
|------|--------|------|------|
| `SEARCH_MAX_CONCURRENCY` | 4 | 1-10 | 同时执行的最大搜索数 |
| `SEARCH_OVERALL_TIMEOUT_SEC` | 10 | 5-60 | 整个搜索过程的最大时间 |
| `SEARCH_PER_FETCH_TIMEOUT_SEC` | 6 | 3-30 | 单个搜索的超时时间 |

## 📊 性能对比

### 串行搜索（优化前）
```
查询1: 今天天气     [2.5秒]
查询2: 科技新闻     [2.8秒]
查询3: 股票市场     [2.2秒]
查询4: 体育赛事     [2.6秒]
--------------------------------
总时间: 10.1秒
```

### 并发搜索（优化后）
```
查询1: 今天天气     [2.5秒] ┐
查询2: 科技新闻     [2.8秒] ├─ 同时执行
查询3: 股票市场     [2.2秒] ├─ 最大并发数: 6
查询4: 体育赛事     [2.6秒] ┘
--------------------------------
总时间: 2.8秒 (减少 72%)
```

## 🔍 执行流程

### 1. 搜索启动
```
🚀 Starting concurrent search for 4 queries (max concurrency: 6)...
⏱️ [Concurrent] Overall timeout: 15s
```

### 2. 并发执行
```
🔍 [Concurrent] Query 1: 今天天气 (topK=3)
🔍 [Concurrent] Query 2: 科技新闻 (topK=3)
🔍 [Concurrent] Query 3: 股票市场 (topK=3)
🔍 [Concurrent] Query 4: 体育赛事 (topK=3)
```

### 3. 结果收集
```
✅ [Concurrent] Query 1 context length: 1234 chars
✅ [Concurrent] Query 2 context length: 987 chars
✅ [Concurrent] Query 3 context length: 1456 chars
✅ [Concurrent] Query 4 context length: 1123 chars
```

### 4. 完成统计
```
🎯 [Concurrent] Search completed: 4 successful, 0 failed
```

## 🛡️ 错误处理

### 超时处理
```
⏰ [Concurrent] Query 3 timed out after 8s
⚠️ [Concurrent] Search terminated due to timeout: 3 successful, 1 failed
```

### 失败重试
```
⚠️ [Concurrent] Query 2 Google failed, falling back to DuckDuckGo: quota exceeded
✅ [Concurrent] Query 2 context length: 987 chars
```

### 部分成功
```
❌ [Concurrent] Query 4 failed: network error
🎯 [Concurrent] Search completed: 3 successful, 1 failed
```

## 🧪 测试方法

### 1. 性能测试
```bash
./test-concurrent-search.sh
```

### 2. 配置测试
```bash
# 测试不同并发数
SEARCH_MAX_CONCURRENCY=2 ./test-concurrent-search.sh
SEARCH_MAX_CONCURRENCY=8 ./test-concurrent-search.sh

# 测试不同超时时间
SEARCH_PER_FETCH_TIMEOUT_SEC=3 ./test-concurrent-search.sh
SEARCH_OVERALL_TIMEOUT_SEC=20 ./test-concurrent-search.sh
```

## 📈 性能优化建议

### 1. 并发数调优
- **低并发** (2-3)：适合网络较慢或服务器资源有限
- **中等并发** (4-6)：平衡性能和资源使用
- **高并发** (7-10)：适合网络快速且服务器资源充足

### 2. 超时时间调优
- **快速网络**：减少超时时间 (3-5秒)
- **慢速网络**：增加超时时间 (8-12秒)
- **不稳定网络**：增加整体超时时间 (15-20秒)

### 3. 监控指标
- 搜索成功率
- 平均响应时间
- 超时频率
- 资源使用率

## 🔧 故障排除

### 常见问题

1. **搜索超时频繁**
   - 增加 `SEARCH_PER_FETCH_TIMEOUT_SEC`
   - 检查网络连接
   - 减少 `SEARCH_MAX_CONCURRENCY`

2. **搜索结果为空**
   - 检查搜索 API 配置
   - 验证 API 密钥和配额
   - 检查网络代理设置

3. **并发数过高导致错误**
   - 减少 `SEARCH_MAX_CONCURRENCY`
   - 检查服务器资源限制
   - 监控内存和 CPU 使用率

## 🎯 最佳实践

1. **渐进式调优**：从默认配置开始，逐步调整参数
2. **监控日志**：关注搜索成功率和响应时间
3. **环境适配**：根据部署环境调整并发数和超时时间
4. **错误处理**：确保部分失败不影响整体功能
5. **资源管理**：避免过度并发导致系统负载过高

通过并发搜索优化，我们显著提升了 web 搜索的响应速度，改善了用户体验，同时保持了系统的稳定性和可靠性。
