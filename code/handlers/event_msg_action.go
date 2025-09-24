package handlers

import (
	"encoding/json"
	"fmt"
	"start-feishubot/services/openai"
)

type MessageAction struct { /*消息*/
}

func (*MessageAction) Execute(a *ActionInfo) bool {
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

	var decision webDecision
	if err := json.Unmarshal([]byte(clsResp.Content), &decision); err != nil {
		// Fallback: if not valid JSON, use original single-shot behavior
		msg := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
		completions, err2 := a.handler.gpt.Completions(msg)
		if err2 != nil {
			replyMsg(*a.ctx, fmt.Sprintf("🤖️：消息机器人摆烂了，请稍后再试～\n错误信息: %v", err2), a.info.msgId)
			return false
		}
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

	if decision.NeedWeb {
		// Step 1 result: needs web – return key query info
		// Compose a concise assistant message with queries
		var payload string
		if len(decision.Queries) > 0 {
			// Join queries into bullet-like lines
			b, _ := json.Marshal(decision.Queries)
			payload = fmt.Sprintf("需要联网检索。请根据以下关键信息进行查询：\n%s", processNewLine(cleanTextBlock(string(b))))
		} else {
			payload = "需要联网检索。请提供更多线索或稍后重试。"
		}
		// Append assistant message to session as the response
		finalHistory := append(history, openai.Messages{Role: "user", Content: a.info.qParsed})
		finalHistory = append(finalHistory, openai.Messages{Role: "assistant", Content: payload})
		a.handler.sessionCache.SetMsg(*a.info.sessionId, finalHistory)
		// new topic card if applicable
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
