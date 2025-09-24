package handlers

import (
	"encoding/json"
	"fmt"
	"start-feishubot/services/openai"
	"start-feishubot/utils"
	"strings"
)

type MessageAction struct { /*消息*/
}

func (*MessageAction) Execute(a *ActionInfo) bool {
	// Handle confirmation command to proceed with web search from previous suggestion
	trimmed := strings.TrimSpace(a.info.qParsed)
	if trimmed == "/search" || trimmed == "继续联网" || trimmed == "继续" {
		history := a.handler.sessionCache.GetMsg(*a.info.sessionId)
		// find last assistant message containing CONFIRM_WEB payload
		var payloadJSON string
		for i := len(history) - 1; i >= 0; i-- {
			if history[i].Role != "assistant" {
				continue
			}
			idx := strings.LastIndex(history[i].Content, "CONFIRM_WEB:")
			if idx >= 0 {
				payloadJSON = strings.TrimSpace(history[i].Content[idx+len("CONFIRM_WEB:"):])
				break
			}
		}
		if payloadJSON == "" {
			// nothing to confirm, continue normal flow
		} else {
			type confirmPayload struct {
				Question string   `json:"question"`
				Queries  []string `json:"queries"`
			}
			var cp confirmPayload
			if err := json.Unmarshal([]byte(payloadJSON), &cp); err == nil {
				// perform the same second-stage logic as auto path
				queries := cp.Queries
				if len(queries) == 0 {
					queries = []string{cp.Question}
				}
				fmt.Println("[Second Stage Confirmed] queries:", queries)
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
					ctx, err := utils.BuildSearchContext(q, 3)
					if err != nil || strings.TrimSpace(ctx) == "" {
						continue
					}
					ctxParts = append(ctxParts, fmt.Sprintf("{\"query\": %q, \"sources\": %s}", q, ctx))
				}
				fmt.Println("[Second Stage Confirmed] built contexts:", len(ctxParts))
				if len(ctxParts) == 0 {
					if err := replyMsg(*a.ctx, "尝试联网检索未获取到有效资料，请稍后再试。", a.info.msgId); err != nil {
						replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
					}
					return false
				}
				contextJSON := "[" + strings.Join(ctxParts, ",") + "]"
				webSystem := openai.Messages{Role: "system", Content: "你是一个联网助手。根据给定的检索资料（JSON 数组，含 query 与 sources 列表，每个 source 有 title、url、content），请严谨回答用户问题：\n- 仅使用资料中能够支持的事实；\n- 不确定时明确说明不确定；\n- 在内容末尾列出引用的网址列表。"}
				userWithCtx := openai.Messages{Role: "user", Content: fmt.Sprintf("用户问题：%s\n检索资料(JSON)：%s", cp.Question, contextJSON)}
				secondMsgs := append(history, webSystem)
				secondMsgs = append(secondMsgs, userWithCtx)
				finalResp, err := a.handler.gpt.Completions(secondMsgs)
				if err != nil {
					replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err), a.info.msgId)
					return false
				}
				fmt.Println("[OpenAI Second] raw:", finalResp.Content)
				finalHistory := append(history, openai.Messages{Role: "user", Content: cp.Question})
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
		}
		// if no payload, fall through to normal flow
	}
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
		// 改为确认流：先把 queries 回显并携带 CONFIRM_WEB 负载，等待用户指令
		var payload string
		if len(decision.Queries) > 0 {
			b, _ := json.Marshal(decision.Queries)
			payload = fmt.Sprintf("检测到该问题可能需要联网检索。\n建议查询关键信息：\n%s\n\n如需继续，请回复 /search 或 继续联网。\nCONFIRM_WEB:%s",
				processNewLine(cleanTextBlock(string(b))),
				fmt.Sprintf("{\"question\": %q, \"queries\": %s}", a.info.qParsed, string(b)))
		} else {
			payload = fmt.Sprintf("检测到该问题可能需要联网检索。\n如需继续，请回复 /search 或 继续联网。\nCONFIRM_WEB:%s",
				fmt.Sprintf("{\"question\": %q, \"queries\": []}", a.info.qParsed))
		}
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
