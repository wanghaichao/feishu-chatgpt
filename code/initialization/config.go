package initialization

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	FeishuAppId                string
	FeishuAppSecret            string
	FeishuAppEncryptKey        string
	FeishuAppVerificationToken string
	FeishuBotName              string
	OpenaiApiKeys              []string
	HttpPort                   int
	HttpsPort                  int
	UseHttps                   bool
	CertFile                   string
	KeyFile                    string
	OpenaiApiUrl               string
	HttpProxy                  string
	// provider switch: "openai" (default) or "ark"
	Provider string
	// Ark (Volcengine Ark Bots) configurations
	ArkApiKey string
	ArkApiUrl string
	ArkBotId  string
	// debug http request/response logs
	DebugHTTP bool
	// Always perform web search before answering
	SearchAlways bool
	// Number of top results to read
	SearchTopK int
	// Search behavior options
	SearchOverallTimeoutSec  int
	SearchPerFetchTimeoutSec int
	SearchMaxConcurrency     int
	SearchCacheTTLMin        int
	// Trigger control
	SearchOnlyOnKeywords bool
	SearchKeywords       []string
	// Google Custom Search API key
	GoogleApiKey string
	// Google Custom Search Engine ID (cx)
	GoogleCSEId string
}

func LoadConfig(cfg string) *Config {
	viper.SetConfigFile(cfg)
	viper.ReadInConfig()
	viper.AutomaticEnv()
	//content, err := ioutil.ReadFile("config.yaml")
	//if err != nil {
	//	fmt.Println("Error reading file:", err)
	//}
	//fmt.Println(string(content))

	config := &Config{
		FeishuAppId:                getViperStringValue("APP_ID", ""),
		FeishuAppSecret:            getViperStringValue("APP_SECRET", ""),
		FeishuAppEncryptKey:        getViperStringValue("APP_ENCRYPT_KEY", ""),
		FeishuAppVerificationToken: getViperStringValue("APP_VERIFICATION_TOKEN", ""),
		FeishuBotName:              getViperStringValue("BOT_NAME", ""),
		OpenaiApiKeys:              getViperStringArray("OPENAI_KEY", nil),
		HttpPort:                   getViperIntValue("HTTP_PORT", 9000),
		HttpsPort:                  getViperIntValue("HTTPS_PORT", 9001),
		UseHttps:                   getViperBoolValue("USE_HTTPS", false),
		CertFile:                   getViperStringValue("CERT_FILE", "cert.pem"),
		KeyFile:                    getViperStringValue("KEY_FILE", "key.pem"),
		OpenaiApiUrl:               getViperStringValue("API_URL", "https://api.openai.com/v1"),
		HttpProxy:                  getViperStringValue("HTTP_PROXY", ""),
		Provider:                   getViperStringValue("PROVIDER", "openai"),
		ArkApiKey:                  getViperStringValue("ARK_API_KEY", ""),
		ArkApiUrl:                  getViperStringValue("ARK_API_URL", "https://ark.cn-beijing.volces.com/api/v3/bots"),
		ArkBotId:                   getViperStringValue("ARK_BOT_ID", ""),
		DebugHTTP:                  getViperBoolValue("DEBUG_HTTP", true),
		SearchAlways:               getViperBoolValue("SEARCH_ALWAYS", true),
		SearchTopK:                 getViperIntValue("SEARCH_TOPK", 3),
		SearchOverallTimeoutSec:    getViperIntValue("SEARCH_OVERALL_TIMEOUT_SEC", 10),
		SearchPerFetchTimeoutSec:   getViperIntValue("SEARCH_PER_FETCH_TIMEOUT_SEC", 6),
		SearchMaxConcurrency:       getViperIntValue("SEARCH_MAX_CONCURRENCY", 4),
		SearchCacheTTLMin:          getViperIntValue("SEARCH_CACHE_TTL_MIN", 5),
		SearchOnlyOnKeywords:       getViperBoolValue("SEARCH_ONLY_ON_KEYWORDS", true),
		SearchKeywords:             getViperStringArray("SEARCH_KEYWORDS", []string{"/read", "联网", "上网", "google", "谷歌", "搜索", "查一下", "最新", "实时"}),
		GoogleApiKey:               getViperStringValue("GOOGLE_API_KEY", ""),
		GoogleCSEId:                getViperStringValue("GOOGLE_CSE_ID", ""),
	}

	return config
}

func getViperStringValue(key string, defaultValue string) string {
	value := viper.GetString(key)
	if value == "" {
		return defaultValue
	}
	return value
}

// OPENAI_KEY: sk-xxx,sk-xxx,sk-xxx
// result:[sk-xxx sk-xxx sk-xxx]
func getViperStringArray(key string, defaultValue []string) []string {
	// 优先读取以逗号分隔的环境变量
	if envVal := os.Getenv(key); envVal != "" {
		parts := strings.Split(envVal, ",")
		var trimmed []string
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p != "" {
				trimmed = append(trimmed, p)
			}
		}
		if len(trimmed) > 0 {
			return trimmed
		}
	}
	// 其次读取配置文件中的数组
	if v := viper.GetStringSlice(key); len(v) > 0 {
		return v
	}
	return defaultValue
}

func getViperIntValue(key string, defaultValue int) int {
	value := viper.GetInt(key)
	if value == 0 {
		return defaultValue
	}
	return value
}

func getViperBoolValue(key string, defaultValue bool) bool {
	if !viper.IsSet(key) {
		return defaultValue
	}
	return viper.GetBool(key)
}

func (config *Config) GetCertFile() string {
	if config.CertFile == "" {
		return "cert.pem"
	}
	if _, err := os.Stat(config.CertFile); err != nil {
		fmt.Printf("Certificate file %s does not exist, using default file cert.pem\n", config.CertFile)
		return "cert.pem"
	}
	return config.CertFile
}

func (config *Config) GetKeyFile() string {
	if config.KeyFile == "" {
		return "key.pem"
	}
	if _, err := os.Stat(config.KeyFile); err != nil {
		fmt.Printf("Key file %s does not exist, using default file key.pem\n", config.KeyFile)
		return "key.pem"
	}
	return config.KeyFile
}

// 过滤出 "sk-" 开头的 key
func filterFormatKey(keys []string) []string {
	var result []string
	for _, key := range keys {
		if strings.HasPrefix(key, "sk-") {
			result = append(result, key)
		}
	}
	return result

}
