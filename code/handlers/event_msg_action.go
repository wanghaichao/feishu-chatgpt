package handlers

import (
	"fmt"
	"start-feishubot/services/types" // 使用通用类型
)

type MessageAction struct{}

func (*MessageAction) Execute(a *ActionInfo) bool {
	// 使用通用类型 types.Message
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	
	// 创建消息结构体
	userMsg := types.Message{
		Role:    "user", 
		Content: a.info.qParsed,
	}
	
	msg = append(msg, userMsg)
	
	// 调用 Completions
	completion, err := a.handler.gpt.Completions(msg)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：DeepSeek 服务暂时不可用，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	
	// 将回复添加到消息历史
	assistantMsg := types.Message{
		Role:    "assistant",
		Content: completion.Content,
	}
	msg = append(msg, assistantMsg)
	
	a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
	
	// 新话题处理
	if len(msg) == 2 {
		sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, completion.Content)
		return false
	}
	
	// 回复消息
	if err := replyMsg(*a.ctx, completion.Content, a.info.msgId); err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息发送失败，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	
	return true
}
