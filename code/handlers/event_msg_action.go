package handlers

import (
	"encoding/json"
	"fmt"
	"start-feishubot/services/openai"
	"start-feishubot/utils"
	"strings"
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
	fmt.Printf("[MessageAction] Starting two-stage flow for: %s\n", a.info.qParsed)
	// Step 1: classification – decide if we need web and extract key queries
	type webDecision struct {
		NeedWeb bool     `json:"need_web"`
		Queries []string `json:"queries,omitempty"`
		Answer  string   `json:"answer,omitempty"`
		Reason  string   `json:"reason,omitempty"`
	}

	// Build classification prompt
	classifySystem := openai.Messages{Role: "system", Content: "你是一个助手。请严格输出 JSON，不要包含多余文本。根据用户问题判断是否需要联网检索外部信息才能给出可靠答案。若需要，请给出3-6条精炼的中文检索关键信息（queries）。若不需要，请直接给出最终答案。必须输出如下 JSON：{\"need_web\": boolean, \"queries\": string[], \"answer\": string}. 当 need_web=true 时，尽量填写 queries，answer 可留空；当 need_web=false 时，必须填写 answer，queries 可留空。"}

	history := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	classifyMsgs := append([]openai.Messages{classifySystem}, history...)
	classifyMsgs = append(classifyMsgs, openai.Messages{Role: "user", Content: a.info.qParsed})

	clsResp, err := a.handler.gpt.Completions(classifyMsgs)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	// debug: print first-stage raw output
	fmt.Println("[OpenAI First] raw:", clsResp.Content)

	var decision webDecision
	if err := json.Unmarshal([]byte(clsResp.Content), &decision); err != nil {
		// Fallback: if not valid JSON, use original single-shot behavior
		msg := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
		completions, err2 := a.handler.gpt.Completions(msg)
		if err2 != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err2), a.info.msgId)
			return false
		}
		// debug: print single-shot raw output
		fmt.Println("[OpenAI Single] raw:", completions.Content)
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
	if b, _ := json.Marshal(decision); len(b) > 0 {
		fmt.Println("[Decision JSON]:", string(b))
	}

	if decision.NeedWeb {
		// Step 2: 自动触发检索与二次回答
		queries := decision.Queries
		if len(queries) == 0 {
			queries = []string{a.info.qParsed}
		}
		fmt.Println("[Second Stage Triggered] queries:", queries)
		// 最多取前三条查询，分别构建搜索上下文
		maxQ := 3
		if len(queries) < maxQ {
			maxQ = len(queries)
		}
		var ctxParts []string
		for i := 0; i < maxQ; i++ {
			q := strings.TrimSpace(queries[i])
			if q == "" {
				continue
			}
			fmt.Printf("[Web Search] Query %d: %s\n", i+1, q)
			ctx, err := utils.BuildSearchContext(q, 3)
			if err != nil {
				fmt.Printf("[Web Search] Query %d failed: %v\n", i+1, err)
				continue
			}
			if strings.TrimSpace(ctx) == "" {
				fmt.Printf("[Web Search] Query %d returned empty context\n", i+1)
				continue
			}
			fmt.Printf("[Web Search] Query %d context length: %d chars\n", i+1, len(ctx))
			fmt.Printf("[Web Search] Query %d context preview: %s...\n", i+1, ctx[:min(200, len(ctx))])
			ctxParts = append(ctxParts, fmt.Sprintf("{\"query\": %q, \"sources\": %s}", q, ctx))
		}
		fmt.Println("[Second Stage] built contexts:", len(ctxParts))
		if len(ctxParts) == 0 {
			// 无法拿到上下文，退化为提示 queries
			var payload string
			if len(decision.Queries) > 0 {
				b, _ := json.Marshal(decision.Queries)
				payload = fmt.Sprintf("需要联网检索。请根据以下关键信息进行查询：\n%s", processNewLine(cleanTextBlock(string(b))))
			} else {
				payload = "需要联网检索，但暂未获取到有效资料。请稍后重试。"
			}
			fmt.Println("[Second Stage] no context, reply with queries")
			finalHistory := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
			finalHistory = append(finalHistory, openai.Messages{Role: "assistant", Content: payload})
			a.handler.sessionCache.SetMsg(*a.info.sessionId, finalHistory)
			if len(finalHistory) == 2 {
				sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, payload)
				return false
			}
			if err := replyMsg(*a.ctx, payload, a.info.msgId); err != nil {
				replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
				return false
			}
			return true
		}
		// 组合检索上下文为 JSON 数组字符串
		contextJSON := "[" + strings.Join(ctxParts, ",") + "]"
		fmt.Printf("[Second Stage] Final context JSON length: %d chars\n", len(contextJSON))
		fmt.Printf("[Second Stage] Final context JSON preview: %s...\n", contextJSON[:min(500, len(contextJSON))])
		// 构建二次提问消息，携带检索资料
		webSystem := openai.Messages{Role: "system", Content: "你是一个联网助手。根据给定的检索资料（JSON 数组，含 query 与 sources 列表，每个 source 有 title、url、content），请严谨回答用户问题：\n- 仅使用资料中能够支持的事实；\n- 不确定时明确说明不确定；\n- 在内容末尾列出引用的网址列表。"}
		userWithCtx := openai.Messages{Role: "user", Content: fmt.Sprintf("用户问题：%s\n检索资料(JSON)：%s", a.info.qParsed, contextJSON)}
		secondMsgs := append(history, webSystem)
		secondMsgs = append(secondMsgs, userWithCtx)
		finalResp, err := a.handler.gpt.Completions(secondMsgs)
		if err != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		// debug: print second-stage raw output
		fmt.Println("[OpenAI Second] raw:", finalResp.Content)
		finalHistory := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
		finalHistory = append(finalHistory, openai.Messages{Role: "assistant", Content: finalResp.Content})
		a.handler.sessionCache.SetMsg(*a.info.sessionId, finalHistory)
		if len(finalHistory) == 2 {
			sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, finalResp.Content)
			return false
		}
		if err := replyMsg(*a.ctx, finalResp.Content, a.info.msgId); err != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
			return false
		}
		return true
	}

	// NeedWeb == false: directly return final answer from decision.Answer
	answer := decision.Answer
	if answer == "" {
		// Safety fallback: run a normal completion to produce an answer
		msg := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
		completions, err2 := a.handler.gpt.Completions(msg)
		if err2 != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err2), a.info.msgId)
			return false
		}
		// debug: print direct-fallback raw output
		fmt.Println("[OpenAI Direct Fallback] raw:", completions.Content)
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
