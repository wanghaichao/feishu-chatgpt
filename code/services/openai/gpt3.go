package openai

import (
	"errors"
)

const (
	maxCompletionTokens = 2000
	engine              = "gpt-5" // 或 "gpt-5-turbo" 取决于你要用的版本
)

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatGPTResponseBody 响应体
type ChatGPTResponseBody struct {
	ID      string                 `json:"id"`
	Object  string                 `json:"object"`
	Created int                    `json:"created"`
	Model   string                 `json:"model"`
	Choices []ChatGPTChoiceItem    `json:"choices"`
	Usage   map[string]interface{} `json:"usage"`
}

type ChatGPTChoiceItem struct {
	Message      Messages `json:"message"`
	Index        int      `json:"index"`
	FinishReason string   `json:"finish_reason"`
}

// ChatGPTRequestBody 请求体（GPT-5 版）
type ChatGPTRequestBody struct {
	Model               string     `json:"model"`
	Messages            []Messages `json:"messages"`
	MaxCompletionTokens int        `json:"max_completion_tokens"`
}

func (gpt ChatGPT) Completions(msg []Messages) (resp Messages, err error) {
	requestBody := ChatGPTRequestBody{
		Model:               engine,
		Messages:            msg,
		MaxCompletionTokens: maxCompletionTokens,
	}

	gptResponseBody := &ChatGPTResponseBody{}
	err = gpt.sendRequestWithBodyType(
		gpt.ApiUrl+"/v1/chat/completions",
		"POST",
		jsonBody, // 假设你已有定义：const jsonBody = "application/json"
		requestBody,
		gptResponseBody,
	)

	if err == nil && len(gptResponseBody.Choices) > 0 {
		resp = gptResponseBody.Choices[0].Message
	} else {
		resp = Messages{}
		err = errors.New("openai 请求失败")
	}
	return resp, err
}
