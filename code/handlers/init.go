package handlers

import (
	"context"
	"fmt"
	"start-feishubot/initialization"
	"start-feishubot/services/openai"

	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"
	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

type MessageHandlerInterface interface {
	msgReceivedHandler(ctx context.Context, event *larkim.P2MessageReceiveV1) error
	cardHandler(ctx context.Context, cardAction *larkcard.CardAction) (interface{}, error)
}

type HandlerType string

const (
	GroupHandler = "group"
	UserHandler  = "personal"
)

// handlers 所有消息类型类型的处理器
var handlers MessageHandlerInterface

func InitHandlers(gpt *openai.ChatGPT, config initialization.Config) {
	handlers = NewMessageHandler(gpt, config)
}

func Handler(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	// 添加空指针检查
	if event == nil || event.Event == nil || event.Event.Message == nil {
		fmt.Println("❌ Handler: Invalid event structure: nil pointer detected")
		return fmt.Errorf("invalid event structure")
	}

	// 安全地获取消息ID
	var msgIdStr string
	if event.Event.Message.MessageId != nil {
		msgIdStr = *event.Event.Message.MessageId
	} else {
		msgIdStr = "unknown"
	}
	fmt.Printf("🎯 Handler called with event: %s\n", msgIdStr)

	// 安全地获取聊天类型和消息类型
	var chatTypeStr, msgTypeStr string
	if event.Event.Message.ChatType != nil {
		chatTypeStr = *event.Event.Message.ChatType
	} else {
		chatTypeStr = "unknown"
	}
	if event.Event.Message.MessageType != nil {
		msgTypeStr = *event.Event.Message.MessageType
	} else {
		msgTypeStr = "unknown"
	}
	fmt.Printf("📋 Event details: chatType=%s, msgType=%s\n", chatTypeStr, msgTypeStr)

	return handlers.msgReceivedHandler(ctx, event)
}

func ReadHandler(ctx context.Context, event *larkim.P2MessageReadV1) error {
	_ = event.Event.Reader.ReaderId.OpenId
	//fmt.Printf("msg is read by : %v \n", *readerId)
	return nil
}

func CardHandler() func(ctx context.Context,
	cardAction *larkcard.CardAction) (interface{}, error) {
	return func(ctx context.Context, cardAction *larkcard.CardAction) (interface{}, error) {
		//handlerType := judgeCardType(cardAction)
		return handlers.cardHandler(ctx, cardAction)
	}
}

func judgeCardType(cardAction *larkcard.CardAction) HandlerType {
	actionValue := cardAction.Action.Value
	chatType := actionValue["chatType"]
	//fmt.Printf("chatType: %v", chatType)
	if chatType == "group" {
		return GroupHandler
	}
	if chatType == "personal" {
		return UserHandler
	}
	return "otherChat"
}

func judgeChatType(event *larkim.P2MessageReceiveV1) HandlerType {
	// 添加空指针检查
	if event == nil || event.Event == nil || event.Event.Message == nil || event.Event.Message.ChatType == nil {
		fmt.Println("❌ judgeChatType: Invalid event structure")
		return "otherChat"
	}

	chatType := *event.Event.Message.ChatType
	if chatType == "group" {
		return GroupHandler
	}
	if chatType == "p2p" {
		return UserHandler
	}
	return "otherChat"
}
