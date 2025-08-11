package openai

import (
	"errors"
)

const (
	maxTokens   = 10000
	engine      = "gpt-5"
)

// ChatGPTResponseBody 请求体
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

// ChatGPTRequestBody 响应体
type ChatGPTRequestBody struct {
	Model       string     `json:"model"`
	Messages    []Messages `json:"messages"`
	MaxTokens   int        `json:"max_tokens"`
}

func (gpt ChatGPT) Completions(msg []Messages) (resp Messages, err error) {
	requestBody := ChatGPTRequestBody{
		Model:       engine,
		Messages:    msg,
		MaxTokens:   maxTokens,
	}
	gptResponseBody := &ChatGPTResponseBody{}
	err = gpt.sendRequestWithBodyType(gpt.ApiUrl+"/v1/chat/completions", "POST",
		jsonBody,
		requestBody, gptResponseBody)

	if err == nil && len(gptResponseBody.Choices) > 0 {
		resp = gptResponseBody.Choices[0].Message
	} else {
		resp = Messages{}
		err = errors.New("openai 请求失败")
	}
	return resp, err
}
