package main

import (
	"context"
	"log"
	"os"
	"start-feishubot/handlers"
	"start-feishubot/initialization"
	"start-feishubot/services/openai"
	"strconv"

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
	log.Println("🚀 Starting Feishu ChatGPT Bot...")

	initialization.InitRoleList()
	pflag.Parse()
	config := initialization.LoadConfig(*cfg)

	// 支持 Railway 的 PORT 环境变量
	if port := os.Getenv("PORT"); port != "" {
		log.Printf("🌐 Using Railway PORT: %s", port)
		// 将 PORT 环境变量转换为整数并设置到配置中
		if portInt, err := strconv.Atoi(port); err == nil {
			config.HttpPort = portInt
			log.Printf("✅ Port set to: %d", config.HttpPort)
		} else {
			log.Printf("❌ Invalid PORT value: %s, using default 9000", port)
		}
	} else {
		log.Printf("📡 Using default port: %d", config.HttpPort)
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

	log.Printf("🎯 Starting server on port %d...", config.HttpPort)
	log.Printf("🔗 Health check available at: http://localhost:%d/ping", config.HttpPort)
	log.Printf("🔗 Webhook endpoint: http://localhost:%d/webhook/event", config.HttpPort)

	err := initialization.StartServer(*config, r)
	if err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}

}
