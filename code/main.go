package main

import (
	"context"
	"log"
	"os"
	"start-feishubot/handlers"
	"start-feishubot/initialization"
	"start-feishubot/services/openai"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"

	larkcard "github.com/larksuite/oapi-sdk-go/v3/card"

	"github.com/gin-gonic/gin"
	"github.com/spf13/pflag"

	sdkginext "github.com/larksuite/oapi-sdk-gin"

	"github.com/larksuite/oapi-sdk-go/v3/event/dispatcher"
)

var (
	cfg = pflag.StringP("config", "c", "./config.yaml", "apiserver config file path.")
)

func main() {
	initialization.InitRoleList()
	pflag.Parse()
	config := initialization.LoadConfig(*cfg)

	// 支持 Railway 的 PORT 环境变量
	if port := os.Getenv("PORT"); port != "" {
		log.Printf("Using Railway PORT: %s", port)
		config.HttpPort = 9000 // Railway 会自动映射到 PORT
	}

	initialization.LoadLarkClient(*config)
	gpt := openai.NewChatGPT(*config)
	handlers.InitHandlers(gpt, *config)

	eventHandler := dispatcher.NewEventDispatcher(
		config.FeishuAppVerificationToken, config.FeishuAppEncryptKey).
		OnP2MessageReceiveV1(handlers.Handler).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			return handlers.ReadHandler(ctx, event)
		})

	cardHandler := larkcard.NewCardActionHandler(
		config.FeishuAppVerificationToken, config.FeishuAppEncryptKey,
		handlers.CardHandler())

	r := gin.Default()

	// 健康检查端点
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "healthy",
		})
	})

	// Railway 健康检查端点
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "Feishu ChatGPT Bot is running",
			"status":  "healthy",
		})
	})

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "feishu-chatgpt",
		})
	})
	r.POST("/webhook/event",
		sdkginext.NewEventHandlerFunc(eventHandler))
	r.POST("/webhook/card",
		sdkginext.NewCardActionHandlerFunc(
			cardHandler))

	err := initialization.StartServer(*config, r)
	if err != nil {
		log.Fatalf("failed to start server: %v", err)
	}

}
