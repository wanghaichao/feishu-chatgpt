package handlers

import (
	"context"
	"fmt"
	"start-feishubot/initialization"
	"start-feishubot/services"
	"start-feishubot/services/openai"
	"strings"

	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

// 责任链
func chain(data *ActionInfo, actions ...Action) bool {
	for i, v := range actions {
		actionName := fmt.Sprintf("%T", v)
		fmt.Printf("  🔧 Action %d: %s\n", i+1, actionName)

		if !v.Execute(data) {
			fmt.Printf("  ⏹️ Action %d (%s) returned false, stopping chain\n", i+1, actionName)
			return false
		}
		fmt.Printf("  ✅ Action %d (%s) completed\n", i+1, actionName)
	}
	return true
}

type MessageHandler struct {
	sessionCache services.SessionServiceCacheInterface
	msgCache     services.MsgCacheInterface
	gpt          *openai.ChatGPT
	config       initialization.Config
}

func (m MessageHandler) cardHandler(ctx context.Context,
	cardAction *larkcard.CardAction) (interface{}, error) {
	messageHandler := NewCardHandler(m)
	return messageHandler(ctx, cardAction)
}

func judgeMsgType(event *larkim.P2MessageReceiveV1) (string, error) {
	msgType := event.Event.Message.MessageType

	switch *msgType {
	case "text", "image", "audio":
		return *msgType, nil
	default:
		return "", fmt.Errorf("unknown message type: %v", *msgType)
	}

}

func (m MessageHandler) msgReceivedHandler(ctx context.Context, event *larkim.P2MessageReceiveV1) error {
	fmt.Printf("📨 Received message event: %s\n", *event.Event.Message.MessageId)

	handlerType := judgeChatType(event)
	fmt.Printf("🔍 Chat type: %s\n", handlerType)
	if handlerType == "otherChat" {
		fmt.Println("❌ Unknown chat type, ignoring")
		return nil
	}

	msgType, err := judgeMsgType(event)
	if err != nil {
		fmt.Printf("❌ Error getting message type: %v\n", err)
		return nil
	}
	fmt.Printf("📝 Message type: %s\n", msgType)

	content := event.Event.Message.Content
	msgId := event.Event.Message.MessageId
	rootId := event.Event.Message.RootId
	chatId := event.Event.Message.ChatId
	mention := event.Event.Message.Mentions

	fmt.Printf("📋 Message details: msgId=%s, chatId=%s\n", *msgId, *chatId)
	if rootId != nil {
		fmt.Printf("🔗 Root ID: %s\n", *rootId)
	}

	sessionId := rootId
	if sessionId == nil || *sessionId == "" {
		sessionId = msgId
		fmt.Printf("🆔 Using msgId as sessionId: %s\n", *sessionId)
	} else {
		fmt.Printf("🆔 Using rootId as sessionId: %s\n", *sessionId)
	}

	parsedContent := strings.Trim(parseContent(*content), " ")
	fmt.Printf("📝 Parsed content: %s\n", parsedContent)

	msgInfo := MsgInfo{
		handlerType: handlerType,
		msgType:     msgType,
		msgId:       msgId,
		chatId:      chatId,
		qParsed:     parsedContent,
		fileKey:     parseFileKey(*content),
		imageKey:    parseImageKey(*content),
		sessionId:   sessionId,
		mention:     mention,
	}
	data := &ActionInfo{
		ctx:     &ctx,
		handler: &m,
		info:    &msgInfo,
	}

	fmt.Println("🔄 Starting action chain...")
	actions := []Action{
		&ProcessedUniqueAction{}, //避免重复处理
		&ProcessMentionAction{},  //判断机器人是否应该被调用
		&AudioAction{},           //语音处理
		&EmptyAction{},           //空消息处理
		&WebBrowseAction{},       //联网读取
		&AutoSearchAction{},      //自动联网搜索
		&ClearAction{},           //清除消息处理
		&PicAction{},             //图片处理
		&RoleListAction{},        //角色列表处理
		&HelpAction{},            //帮助处理
		&BalanceAction{},         //余额处理
		&RolePlayAction{},        //角色扮演处理
		&MessageAction{},         //消息处理
	}

	fmt.Printf("📋 Executing %d actions in chain\n", len(actions))
	chain(data, actions...)
	fmt.Println("✅ Action chain completed")
	return nil
}

var _ MessageHandlerInterface = (*MessageHandler)(nil)

func NewMessageHandler(gpt *openai.ChatGPT,
	config initialization.Config) MessageHandlerInterface {
	return &MessageHandler{
		sessionCache: services.GetSessionCache(),
		msgCache:     services.GetMsgCache(),
		gpt:          gpt,
		config:       config,
	}
}

func (m MessageHandler) judgeIfMentionMe(mention []*larkim.
	MentionEvent) bool {
	if len(mention) != 1 {
		return false
	}
	return *mention[0].Name == m.config.FeishuBotName
}
