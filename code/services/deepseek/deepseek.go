package deepseek

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

const (
	volcEngineURL = "https://ark.cn-beijing.volces.com/api/v3/bots/chat/completions"
)

// 保持与 OpenAI 相同的结构体名称
type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// 保持与 OpenAI 相同的结构体名称
type GPT struct {
	ApiKey string
	Engine string // 添加引擎字段
	Client *http.Client
}

func NewGPT(apiKey, engine string) *GPT {
	return &GPT{
		ApiKey: apiKey,
		Engine: engine,
		Client: &http.Client{
			Timeout: 8 * time.Second, // 保持飞书要求的快速响应
		},
	}
}

// 保持与 OpenAI 相同的函数签名
func (gpt *GPT) Completions(msg []Messages) (resp Messages, err error) {
	requestBody := struct {
		Model    string     `json:"model"`
		Messages []Messages `json:"messages"`
		Stream   bool       `json:"stream"`
	}{
		Model:    gpt.Engine, // 使用配置的引擎
		Messages: msg,
		Stream:   false, // 必须关闭流式
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return Messages{}, err
	}

	req, err := http.NewRequest("POST", volcEngineURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return Messages{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gpt.ApiKey)

	respHttp, err := gpt.Client.Do(req)
	if err != nil {
		return Messages{}, err
	}
	defer respHttp.Body.Close()

	// 处理非200状态码
	if respHttp.StatusCode != http.StatusOK {
		var errorResponse struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		
		if err := json.NewDecoder(respHttp.Body).Decode(&errorResponse); err == nil {
			return Messages{}, errors.New("DeepSeek API: " + errorResponse.Error.Message)
		}
		return Messages{}, errors.New("DeepSeek API error: " + respHttp.Status)
	}

	var responseBody struct {
		Choices []struct {
			Message Messages `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(respHttp.Body).Decode(&responseBody); err != nil {
		return Messages{}, err
	}

	if len(responseBody.Choices) > 0 {
		return responseBody.Choices[0].Message, nil
	}

	return Messages{}, errors.New("empty response from DeepSeek")
}
