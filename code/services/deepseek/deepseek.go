package deepseek

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"start-feishubot/services/types" // 引入通用类型
	"time"
)

const (
	volcEngineURL = "https://ark.cn-beijing.volces.com/api/v3/bots/chat/completions"
)

type GPT struct {
	ApiKey string
	Engine string
	Client *http.Client
}

func NewGPT(apiKey, engine string) *GPT {
	return &GPT{
		ApiKey: apiKey,
		Engine: engine,
		Client: &http.Client{
			Timeout: 8 * time.Second,
		},
	}
}

func (gpt *GPT) Completions(msg []types.Message) (types.Message, error) {
	requestBody := struct {
		Model    string          `json:"model"`
		Messages []types.Message `json:"messages"`
		Stream   bool            `json:"stream"`
	}{
		Model:    gpt.Engine,
		Messages: msg,
		Stream:   false,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return types.Message{}, err
	}

	req, err := http.NewRequest("POST", volcEngineURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return types.Message{}, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gpt.ApiKey)

	respHttp, err := gpt.Client.Do(req)
	if err != nil {
		return types.Message{}, err
	}
	defer respHttp.Body.Close()

	if respHttp.StatusCode != http.StatusOK {
		return types.Message{}, errors.New("DeepSeek API error: " + respHttp.Status)
	}

	var responseBody types.ChatCompletionResponse
	if err := json.NewDecoder(respHttp.Body).Decode(&responseBody); err != nil {
		return types.Message{}, err
	}

	if len(responseBody.Choices) > 0 {
		return responseBody.Choices[0].Message, nil
	}

	return types.Message{}, errors.New("empty response from DeepSeek")
}
