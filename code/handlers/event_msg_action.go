package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"start-feishubot/services/openai"
	"start-feishubot/utils"
	"strings"
	"sync"
	"time"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type MessageAction struct { /*消息*/
}

func (*MessageAction) Execute(a *ActionInfo) bool {
	fmt.Printf("    🔍 MessageAction: Starting two-stage flow for: '%s'\n", a.info.qParsed)
	fmt.Printf("    📋 Session ID: %s\n", *a.info.sessionId)

	// Step 1: classification – decide if we need web and extract key queries
	type webDecision struct {
		NeedWeb    bool     `json:"need_web"`
		Queries    []string `json:"queries,omitempty"`
		Answer     string   `json:"answer,omitempty"`
		Reason     string   `json:"reason,omitempty"`
		SearchTopK int      `json:"search_top_k,omitempty"` // ChatGPT 建议的搜索数量
		MaxTokens  int      `json:"max_tokens,omitempty"`   // ChatGPT 建议的最大token数
	}

	fmt.Printf("    🎯 Step 1: Building classification prompt...\n")
	// Build classification prompt
	classifySystem := openai.Messages{Role: "system", Content: "你是一个助手。请严格输出 JSON，不要包含多余文本。根据用户问题判断是否需要联网检索外部信息才能给出可靠答案。若需要，请给出3-6条精炼的中文检索关键信息（queries），并建议每个查询的搜索数量（search_top_k，建议1-5个结果）和回答的最大token数（max_tokens，建议500-2000）。若不需要，请直接给出最终答案。必须输出如下 JSON：{\"need_web\": boolean, \"queries\": string[], \"answer\": string, \"search_top_k\": number, \"max_tokens\": number}. 当 need_web=true 时，尽量填写 queries、search_top_k 和 max_tokens，answer 可留空；当 need_web=false 时，必须填写 answer 和 max_tokens，queries 和 search_top_k 可留空。"}

	fmt.Printf("    📚 Getting session history...\n")
	history := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	fmt.Printf("    📖 Session history length: %d messages\n", len(history))

	fmt.Printf("    🔧 Building classification messages...\n")
	classifyMsgs := append([]openai.Messages{classifySystem}, history...)
	classifyMsgs = append(classifyMsgs, openai.Messages{Role: "user", Content: a.info.qParsed})
	fmt.Printf("    📝 Total messages to send: %d\n", len(classifyMsgs))

	fmt.Printf("    🤖 Calling OpenAI for classification...\n")
	clsResp, err := a.handler.gpt.Completions(classifyMsgs)
	if err != nil {
		fmt.Printf("    ❌ OpenAI classification failed: %v\n", err)
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}

	fmt.Printf("    ✅ OpenAI classification completed\n")
	fmt.Printf("    📄 Raw response: %s\n", clsResp.Content)

	fmt.Printf("    🔍 Parsing classification result...\n")
	var decision webDecision
	if err := json.Unmarshal([]byte(clsResp.Content), &decision); err != nil {
		fmt.Printf("    ❌ Failed to parse JSON: %v\n", err)
		fmt.Printf("    🔄 Falling back to single-shot behavior...\n")

		// Fallback: if not valid JSON, use original single-shot behavior
		msg := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
		fmt.Printf("    🤖 Calling OpenAI for single-shot response...\n")
		completions, err2 := a.handler.gpt.Completions(msg)
		if err2 != nil {
			fmt.Printf("    ❌ Single-shot OpenAI call failed: %v\n", err2)
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err2), a.info.msgId)
			return false
		}
		fmt.Printf("    ✅ Single-shot response received\n")
		fmt.Printf("    📄 Single-shot raw: %s\n", completions.Content)
		// append to history as final answer
		msg = append(msg, completions)
		a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
		// new topic card logic
		if len(msg) == 2 {
			sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, completions.Content)
			return false
		}
		if err = replyMsg(*a.ctx, completions.Content, a.info.msgId); err != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		return true
	}

	fmt.Printf("    ✅ Classification parsed successfully\n")
	if b, _ := json.Marshal(decision); len(b) > 0 {
		fmt.Printf("    📊 Decision: %s\n", string(b))
	}
	fmt.Printf("    🔍 Decision details: need_web=%t, queries_count=%d, search_top_k=%d, max_tokens=%d\n",
		decision.NeedWeb, len(decision.Queries), decision.SearchTopK, decision.MaxTokens)

	if decision.NeedWeb {
		fmt.Printf("    🌐 Step 2: Web search required\n")
		// Step 2: 自动触发检索与二次回答
		queries := decision.Queries
		if len(queries) == 0 {
			fmt.Printf("    🔄 No queries provided, using original question\n")
			queries = []string{a.info.qParsed}
		}
		fmt.Printf("    🔍 Search queries: %v\n", queries)

		// 使用 ChatGPT 建议的搜索数量，如果没有则使用默认值
		searchTopK := decision.SearchTopK
		if searchTopK <= 0 {
			searchTopK = 3 // 默认值
		}
		if searchTopK > 10 {
			searchTopK = 10 // 限制最大值
		}
		fmt.Printf("[Second Stage] Using SearchTopK: %d\n", searchTopK)

		// 并发搜索：最多取前10条查询，并发构建搜索上下文
		maxQ := 10
		if len(queries) < maxQ {
			maxQ = len(queries)
		}

		// 获取并发数配置
		maxConcurrency := a.handler.config.SearchMaxConcurrency
		if maxConcurrency <= 0 {
			maxConcurrency = 3 // 默认并发数
		}
		if maxConcurrency > 10 {
			maxConcurrency = 10 // 限制最大并发数
		}

		fmt.Printf("🚀 Starting concurrent search for %d queries (max concurrency: %d)...\n", maxQ, maxConcurrency)

		// 创建结果通道和信号量
		type searchResult struct {
			index int
			query string
			ctx   string
			err   error
		}

		resultChan := make(chan searchResult, maxQ)
		semaphore := make(chan struct{}, maxConcurrency) // 信号量控制并发数
		var wg sync.WaitGroup

		// 启动并发搜索
		for i := 0; i < maxQ; i++ {
			q := strings.TrimSpace(queries[i])
			if q == "" {
				continue
			}

			wg.Add(1)
			go func(index int, query string) {
				defer wg.Done()

				// 获取信号量
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				fmt.Printf("🔍 [Concurrent] Query %d: %s (topK=%d)\n", index+1, query, searchTopK)

				// 创建带超时的上下文
				timeout := time.Duration(a.handler.config.SearchPerFetchTimeoutSec) * time.Second
				if timeout <= 0 {
					timeout = 6 * time.Second // 默认超时时间
				}

				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				defer cancel()

				// 使用通道来接收搜索结果
				type searchResultChan struct {
					ctx string
					err error
				}

				searchChan := make(chan searchResultChan, 1)

				go func() {
					var resultCtx string
					var resultErr error

					// 优先使用 Google 搜索，如果失败则回退到 DuckDuckGo
					if a.handler.config.GoogleApiKey != "" && a.handler.config.GoogleCSEId != "" {
						resultCtx, resultErr = utils.BuildGoogleSearchContext(query, a.handler.config.GoogleApiKey, a.handler.config.GoogleCSEId, searchTopK)
						if resultErr != nil {
							fmt.Printf("⚠️ [Concurrent] Query %d Google failed, falling back to DuckDuckGo: %v\n", index+1, resultErr)
							resultCtx, resultErr = utils.BuildSearchContext(query, searchTopK)
						}
					} else {
						resultCtx, resultErr = utils.BuildSearchContext(query, searchTopK)
					}

					searchChan <- searchResultChan{ctx: resultCtx, err: resultErr}
				}()

				// 等待搜索结果或超时
				select {
				case result := <-searchChan:
					// 发送结果到通道
					resultChan <- searchResult{
						index: index,
						query: query,
						ctx:   result.ctx,
						err:   result.err,
					}
				case <-ctx.Done():
					fmt.Printf("⏰ [Concurrent] Query %d timed out after %v\n", index+1, timeout)
					resultChan <- searchResult{
						index: index,
						query: query,
						ctx:   "",
						err:   fmt.Errorf("search timeout after %v", timeout),
					}
				}
			}(i, q)
		}

		// 等待所有搜索完成，带整体超时
		done := make(chan struct{})
		go func() {
			wg.Wait()
			close(resultChan)
			close(done)
		}()

		// 设置整体超时
		overallTimeout := time.Duration(a.handler.config.SearchOverallTimeoutSec) * time.Second
		if overallTimeout <= 0 {
			overallTimeout = 10 * time.Second // 默认整体超时时间
		}

		fmt.Printf("⏱️ [Concurrent] Overall timeout: %v\n", overallTimeout)

		// 收集结果
		var ctxParts []string
		var successfulSearches int
		var failedSearches int
		var timeoutOccurred bool

		// 使用 select 等待结果或超时
		for {
			select {
			case result, ok := <-resultChan:
				if !ok {
					// 所有搜索完成
					fmt.Printf("🎯 [Concurrent] Search completed: %d successful, %d failed\n", successfulSearches, failedSearches)
					goto searchComplete
				}

				if result.err != nil {
					fmt.Printf("❌ [Concurrent] Query %d failed: %v\n", result.index+1, result.err)
					failedSearches++
					continue
				}

				if strings.TrimSpace(result.ctx) == "" {
					fmt.Printf("⚠️ [Concurrent] Query %d returned empty context\n", result.index+1)
					failedSearches++
					continue
				}

				fmt.Printf("✅ [Concurrent] Query %d context length: %d chars\n", result.index+1, len(result.ctx))
				fmt.Printf("📄 [Concurrent] Query %d context preview: %s...\n", result.index+1, result.ctx[:min(200, len(result.ctx))])

				// 确保 ctx 是有效的 JSON 字符串
				var ctxJSON interface{}
				if err := json.Unmarshal([]byte(result.ctx), &ctxJSON); err != nil {
					fmt.Printf("⚠️ [Concurrent] Query %d ctx is not valid JSON, using fallback: %v\n", result.index+1, err)
					ctxParts = append(ctxParts, fmt.Sprintf("{\"query\": %q, \"sources\": \"搜索失败，无法获取内容\"}", result.query))
				} else {
					ctxParts = append(ctxParts, fmt.Sprintf("{\"query\": %q, \"sources\": %s}", result.query, result.ctx))
				}
				successfulSearches++

			case <-time.After(overallTimeout):
				fmt.Printf("⏰ [Concurrent] Overall search timeout after %v\n", overallTimeout)
				timeoutOccurred = true
				goto searchComplete
			}
		}

	searchComplete:
		if timeoutOccurred {
			fmt.Printf("⚠️ [Concurrent] Search terminated due to timeout: %d successful, %d failed\n", successfulSearches, failedSearches)
		} else {
			fmt.Printf("🎯 [Concurrent] Search completed: %d successful, %d failed\n", successfulSearches, failedSearches)
		}
		fmt.Println("[Second Stage] built contexts:", len(ctxParts))

		// 容错处理：即使部分搜索失败，只要有成功的就继续
		if len(ctxParts) == 0 {
			fmt.Printf("⚠️ [Second Stage] No successful searches, but continuing with ChatGPT anyway\n")
			// 即使没有搜索上下文，也继续向 ChatGPT 提问，让它基于自己的知识回答
			ctxParts = []string{"{\"query\": \"用户问题\", \"sources\": \"基于现有知识回答\"}"}
		} else {
			fmt.Printf("✅ [Second Stage] Using %d successful search results, ignoring %d failed searches\n", len(ctxParts), failedSearches)
		}
		// 组合检索上下文为 JSON 数组字符串
		contextJSON := "[" + strings.Join(ctxParts, ",") + "]"
		fmt.Printf("[Second Stage] Final context JSON length: %d chars\n", len(contextJSON))
		fmt.Printf("[Second Stage] Final context JSON preview: %s...\n", contextJSON[:min(500, len(contextJSON))])

		// 验证 JSON 格式
		var jsonTest interface{}
		if err := json.Unmarshal([]byte(contextJSON), &jsonTest); err != nil {
			fmt.Printf("❌ [Second Stage] Invalid JSON format: %v\n", err)
			fmt.Printf("❌ [Second Stage] Raw contextJSON: %s\n", contextJSON)
			// 使用安全的默认值
			contextJSON = "[{\"query\": \"用户问题\", \"sources\": \"基于现有知识回答\"}]"
			fmt.Printf("✅ [Second Stage] Using safe fallback JSON: %s\n", contextJSON)
		} else {
			fmt.Printf("✅ [Second Stage] JSON format is valid\n")
		}
		// 构建二次提问消息，携带检索资料
		webSystem := openai.Messages{Role: "system", Content: "你是一个联网助手。根据给定的检索资料（JSON 数组，含 query 与 sources 列表，每个 source 有 title、url、content），请严谨回答用户问题：\n- 优先使用检索到的资料信息\n- 如果检索资料不足或为空，请基于你的知识库尽力回答\n- 如果某些搜索失败，请基于成功的搜索结果和你的知识给出最佳答案\n- 不确定时明确说明不确定；\n- 在内容末尾列出引用的网址列表（如果有的话）。"}
		userWithCtx := openai.Messages{Role: "user", Content: fmt.Sprintf("用户问题：%s\n检索资料(JSON)：%s", a.info.qParsed, contextJSON)}
		secondMsgs := append(history, webSystem)
		secondMsgs = append(secondMsgs, userWithCtx)

		// 使用 ChatGPT 建议的 max_tokens
		maxTokens := decision.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 1500 // 默认值
		}
		if maxTokens < 100 {
			maxTokens = 500 // 最小值
		}
		if maxTokens > 4000 {
			maxTokens = 4000 // 限制最大值
		}
		fmt.Printf("    🎯 Using ChatGPT suggested max_tokens: %d\n", maxTokens)

		finalResp, err := a.handler.gpt.CompletionsWithMaxTokens(secondMsgs, maxTokens)
		if err != nil {
			fmt.Printf("    ❌ Second stage OpenAI call failed: %v\n", err)
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}

		fmt.Printf("    ✅ Second stage OpenAI call successful\n")
		fmt.Printf("    📄 Response content length: %d\n", len(finalResp.Content))
		fmt.Printf("    📄 Response content: %s\n", finalResp.Content)

		// 检查响应是否为空，如果为空则重试
		if strings.TrimSpace(finalResp.Content) == "" {
			fmt.Printf("    ⚠️ Second stage response is empty, retrying with higher max_tokens...\n")
			maxTokens = maxTokens * 2
			if maxTokens > 4000 {
				maxTokens = 4000
			}
			fmt.Printf("    🔄 Retrying with max_tokens: %d\n", maxTokens)

			finalResp, err = a.handler.gpt.CompletionsWithMaxTokens(secondMsgs, maxTokens)
			if err != nil {
				fmt.Printf("    ❌ Retry failed: %v\n", err)
				replyMsg(*a.ctx, "🤖️：抱歉，我无法生成有效的回答，请稍后再试。", a.info.msgId)
				return false
			}

			if strings.TrimSpace(finalResp.Content) == "" {
				fmt.Printf("    ❌ Retry also returned empty response, trying fallback approach...\n")

				// 尝试使用更简单的提示词和更高的 max_tokens
				simpleSystem := openai.Messages{Role: "system", Content: "你是一个友好的助手。请简洁地回答用户的问题。"}
				simpleUser := openai.Messages{Role: "user", Content: a.info.qParsed}
				simpleMsgs := []openai.Messages{simpleSystem, simpleUser}

				fmt.Printf("    🔄 Trying simple approach with max_tokens: 2000\n")
				finalResp, err = a.handler.gpt.CompletionsWithMaxTokens(simpleMsgs, 2000)

				if err != nil {
					fmt.Printf("    ❌ Simple approach also failed: %v\n", err)
					replyMsg(*a.ctx, "🤖️：抱歉，我暂时无法回答您的问题。请稍后再试或尝试重新表述您的问题。", a.info.msgId)
					return false
				}

				if strings.TrimSpace(finalResp.Content) == "" {
					fmt.Printf("    ❌ Simple approach also returned empty response\n")
					replyMsg(*a.ctx, "🤖️：抱歉，我暂时无法回答您的问题。这可能是因为问题过于复杂或需要更多上下文信息。请尝试重新表述您的问题。", a.info.msgId)
					return false
				}

				fmt.Printf("    ✅ Simple approach successful, got response: %s\n", finalResp.Content[:min(100, len(finalResp.Content))])
			} else {
				fmt.Printf("    ✅ Retry successful, got response: %s\n", finalResp.Content[:min(100, len(finalResp.Content))])
			}
		}
		finalHistory := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
		finalHistory = append(finalHistory, openai.Messages{Role: "assistant", Content: finalResp.Content})
		a.handler.sessionCache.SetMsg(*a.info.sessionId, finalHistory)
		if len(finalHistory) == 2 {
			sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, finalResp.Content)
			return false
		}
		fmt.Printf("    📤 Sending response to user...\n")
		if err := replyMsg(*a.ctx, finalResp.Content, a.info.msgId); err != nil {
			fmt.Printf("    ❌ Failed to send response: %v\n", err)
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		fmt.Printf("    ✅ Response sent successfully\n")
		return true
	}

	// NeedWeb == false: directly return final answer from decision.Answer
	answer := decision.Answer
	if answer == "" {
		// Safety fallback: run a normal completion to produce an answer
		msg := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})

		// 使用 ChatGPT 建议的 max_tokens
		maxTokens := decision.MaxTokens
		if maxTokens <= 0 {
			maxTokens = 1500 // 默认值
		}
		if maxTokens < 100 {
			maxTokens = 500 // 最小值
		}
		if maxTokens > 4000 {
			maxTokens = 4000 // 限制最大值
		}
		fmt.Printf("    🎯 Using ChatGPT suggested max_tokens for fallback: %d\n", maxTokens)

		completions, err2 := a.handler.gpt.CompletionsWithMaxTokens(msg, maxTokens)
		if err2 != nil {
			fmt.Printf("    ❌ Fallback OpenAI call failed: %v\n", err2)
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err2), a.info.msgId)
			return false
		}

		fmt.Printf("    ✅ Fallback OpenAI call successful\n")
		fmt.Printf("    📄 Fallback response content length: %d\n", len(completions.Content))
		fmt.Printf("    📄 Fallback response content: %s\n", completions.Content)

		// 检查响应是否为空，如果为空则重试
		if strings.TrimSpace(completions.Content) == "" {
			fmt.Printf("    ⚠️ Fallback response is empty, retrying with higher max_tokens...\n")
			maxTokens = maxTokens * 2
			if maxTokens > 4000 {
				maxTokens = 4000
			}
			fmt.Printf("    🔄 Retrying fallback with max_tokens: %d\n", maxTokens)

			completions, err2 = a.handler.gpt.CompletionsWithMaxTokens(msg, maxTokens)
			if err2 != nil {
				fmt.Printf("    ❌ Fallback retry failed: %v\n", err2)
				replyMsg(*a.ctx, "🤖️：抱歉，我无法生成有效的回答，请稍后再试。", a.info.msgId)
				return false
			}

			if strings.TrimSpace(completions.Content) == "" {
				fmt.Printf("    ❌ Fallback retry also returned empty response, trying simple approach...\n")

				// 尝试使用最简单的提示词
				simpleMsgs := []openai.Messages{
					{Role: "system", Content: "你是一个友好的助手。"},
					{Role: "user", Content: a.info.qParsed},
				}

				fmt.Printf("    🔄 Trying simple fallback with max_tokens: 2000\n")
				completions, err2 = a.handler.gpt.CompletionsWithMaxTokens(simpleMsgs, 2000)

				if err2 != nil {
					fmt.Printf("    ❌ Simple fallback also failed: %v\n", err2)
					replyMsg(*a.ctx, "🤖️：抱歉，我暂时无法回答您的问题。请稍后再试或尝试重新表述您的问题。", a.info.msgId)
					return false
				}

				if strings.TrimSpace(completions.Content) == "" {
					fmt.Printf("    ❌ Simple fallback also returned empty response\n")
					replyMsg(*a.ctx, "🤖️：抱歉，我暂时无法回答您的问题。这可能是因为问题过于复杂或需要更多上下文信息。请尝试重新表述您的问题。", a.info.msgId)
					return false
				}

				fmt.Printf("    ✅ Simple fallback successful, got response: %s\n", completions.Content[:min(100, len(completions.Content))])
			} else {
				fmt.Printf("    ✅ Fallback retry successful, got response: %s\n", completions.Content[:min(100, len(completions.Content))])
			}
		}
		msg = append(msg, completions)
		a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
		if len(msg) == 2 {
			sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, completions.Content)
			return false
		}
		if err = replyMsg(*a.ctx, completions.Content, a.info.msgId); err != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		return true
	}
	// debug: print direct answer
	fmt.Println("[OpenAI Direct Answer]:", answer)

	// Append assistant answer to history and reply
	finalHistory := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
	finalHistory = append(finalHistory, openai.Messages{Role: "assistant", Content: answer})
	a.handler.sessionCache.SetMsg(*a.info.sessionId, finalHistory)
	if len(finalHistory) == 2 {
		sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, answer)
		return false
	}
	if err := replyMsg(*a.ctx, answer, a.info.msgId); err != nil {
		replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	return true
}
