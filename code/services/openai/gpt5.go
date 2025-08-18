package openai

import (
	"errors"
	"fmt"
	"strings"
)

const (
	maxTokens = 10000
	engine    = "gpt-5"
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
	Model     string     `json:"model"`
	Messages  []Messages `json:"messages"`
	MaxTokens int        `json:"max_completion_tokens"`
}

type ArkOpenAICompatRequestBody struct {
	Model    string     `json:"model"`
	Messages []Messages `json:"messages"`
}

type ArkBotRequestBody struct {
	Input ArkInput `json:"input"`
}

type ArkInput struct {
	Messages []Messages `json:"messages"`
}

type ArkBotResponseBody struct {
	ID     string `json:"id"`
	Output struct {
		Choices []struct {
			Message Messages `json:"message"`
		} `json:"choices"`
	} `json:"output"`
}

func (gpt ChatGPT) Completions(msg []Messages) (resp Messages, err error) {
	// Ark provider branch
	if gpt.Provider == "ark" {
		if gpt.ArkApiUrl == "" || gpt.ArkBotId == "" {
			return Messages{}, errors.New("ark api url or bot id is empty")
		}
		base := strings.TrimRight(gpt.ArkApiUrl, "/")
		if !strings.Contains(base, "/bots") {
			base = base + "/bots"
		}
		// 1) 优先走 OpenAI 兼容路径: /bots/chat/completions，body 为 {model, messages}
		endpointA := fmt.Sprintf("%s/chat/completions", base)
		compatReq := ArkOpenAICompatRequestBody{Model: gpt.ArkBotId, Messages: msg}
		compatResp := &ChatGPTResponseBody{}
		err = gpt.sendRequestWithBodyType(endpointA, "POST", jsonBody, compatReq, compatResp)
		if err == nil && len(compatResp.Choices) > 0 {
			return compatResp.Choices[0].Message, nil
		}
		// 2) 失败则回退到 /bots/{botId}/completions，body 为 {input:{messages}}
		endpointB := fmt.Sprintf("%s/%s/completions", base, gpt.ArkBotId)
		botReq := ArkBotRequestBody{Input: ArkInput{Messages: msg}}
		botResp := &ArkBotResponseBody{}
		err = gpt.sendRequestWithBodyType(endpointB, "POST", jsonBody, botReq, botResp)
		if err == nil && len(botResp.Output.Choices) > 0 {
			return botResp.Output.Choices[0].Message, nil
		}
		return Messages{}, errors.New("ark 请求失败")
	}

	requestBody := ChatGPTRequestBody{
		Model:     engine,
		Messages:  msg,
		MaxTokens: maxTokens,
	}
	gptResponseBody := &ChatGPTResponseBody{}
	err = gpt.sendRequestWithBodyType(gpt.ApiUrl+"/chat/completions", "POST",
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
