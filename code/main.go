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

	log.Println("📋 Initializing role list...")
	initialization.InitRoleList()

	log.Println("⚙️ Parsing command line flags...")
	pflag.Parse()
	log.Printf("📁 Config file: %s", *cfg)

	log.Println("🔧 Loading configuration...")
	config := initialization.LoadConfig(*cfg)
	log.Printf("✅ Config loaded: HTTP_PORT=%d, HTTPS_PORT=%d, USE_HTTPS=%t",
		config.HttpPort, config.HttpsPort, config.UseHttps)

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

	log.Println("🔗 Loading Lark client...")
	initialization.LoadLarkClient(*config)
	log.Printf("✅ Lark client loaded: APP_ID=%s, BOT_NAME=%s",
		config.FeishuAppId, config.FeishuBotName)

	log.Println("🤖 Initializing ChatGPT client...")
	gpt := openai.NewChatGPT(*config)
	log.Printf("✅ ChatGPT client initialized: API_URL=%s, PROVIDER=%s",
		config.OpenaiApiUrl, config.Provider)

	log.Println("🎯 Initializing handlers...")
	handlers.InitHandlers(gpt, *config)
	log.Println("✅ Handlers initialized")

	log.Println("📨 Setting up event dispatcher...")
	eventHandler := dispatcher.NewEventDispatcher(
		config.FeishuAppVerificationToken, config.FeishuAppEncryptKey).
		OnP2MessageReceiveV1(handlers.Handler).
		OnP2MessageReadV1(func(ctx context.Context, event *larkim.P2MessageReadV1) error {
			return handlers.ReadHandler(ctx, event)
		})
	log.Printf("✅ Event dispatcher configured: VERIFICATION_TOKEN=%s",
		config.FeishuAppVerificationToken[:8]+"...")

	log.Println("🎴 Setting up card action handler...")
	cardHandler := larkcard.NewCardActionHandler(
		config.FeishuAppVerificationToken, config.FeishuAppEncryptKey,
		handlers.CardHandler())
	log.Println("✅ Card action handler configured")

	log.Println("🌐 Setting up Gin router...")
	r := gin.Default()

	log.Println("🛣️ Setting up routes...")

	// 健康检查端点
	log.Println("  📍 Registering /ping endpoint")
	r.GET("/ping", func(c *gin.Context) {
		log.Printf("🏥 Health check request from %s", c.ClientIP())
		c.JSON(200, gin.H{
			"message": "pong",
			"status":  "healthy",
		})
	})

	// Railway 健康检查端点
	log.Println("  📍 Registering / endpoint")
	r.GET("/", func(c *gin.Context) {
		log.Printf("🏠 Root request from %s", c.ClientIP())
		c.JSON(200, gin.H{
			"message": "Feishu ChatGPT Bot is running",
			"status":  "healthy",
		})
	})

	// 健康检查端点
	log.Println("  📍 Registering /health endpoint")
	r.GET("/health", func(c *gin.Context) {
		log.Printf("💚 Health check request from %s", c.ClientIP())
		c.JSON(200, gin.H{
			"status":  "healthy",
			"service": "feishu-chatgpt",
		})
	})

	log.Println("  📍 Registering /webhook/event endpoint")
	r.POST("/webhook/event",
		sdkginext.NewEventHandlerFunc(eventHandler))

	log.Println("  📍 Registering /webhook/card endpoint")
	r.POST("/webhook/card",
		sdkginext.NewCardActionHandlerFunc(
			cardHandler))

	log.Println("✅ All routes registered")

	log.Printf("🎯 Starting server on port %d...", config.HttpPort)
	log.Printf("🔗 Health check available at: http://localhost:%d/ping", config.HttpPort)
	log.Printf("🔗 Webhook endpoint: http://localhost:%d/webhook/event", config.HttpPort)

	err := initialization.StartServer(*config, r)
	if err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}

}
