package handlers

import (
	"fmt"
	"start-feishubot/services/deepseek" // 修改导入路径
)

type MessageAction struct { /*消息*/ }

func (*MessageAction) Execute(a *ActionInfo) bool {
	msg := a.handler.sessionCache.GetMsg(*a.info.sessionId)
	
	// 创建消息结构体（保持原结构）
	userMsg := deepseek.Messages{
		Role: "user", 
		Content: a.info.qParsed,
	}
	
	msg = append(msg, userMsg)
	
	// 调用 DeepSeek（保持原方法名）
	completions, err := a.handler.gpt.Completions(msg)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：DeepSeek 服务暂时不可用，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	
	// 将回复添加到消息历史
	assistantMsg := deepseek.Messages{
		Role:    "assistant",
		Content: completions.Content,
	}
	msg = append(msg, assistantMsg)
	
	a.handler.sessionCache.SetMsg(*a.info.sessionId, msg)
	
	// 新话题处理（保持原逻辑）
	if len(msg) == 2 {
		sendNewTopicCard(*a.ctx, a.info.sessionId, a.info.msgId, completions.Content)
		return false
	}
	
	// 回复消息
	err = replyMsg(*a.ctx, completions.Content, a.info.msgId)
	if err != nil {
		replyMsg(*a.ctx, fmt.Sprintf(
			"🤖️：消息发送失败，请稍后再试～\n错误信息: %v", err), a.info.msgId)
		return false
	}
	
	return true
}
