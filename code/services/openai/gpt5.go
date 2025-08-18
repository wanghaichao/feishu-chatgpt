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
		requestBody := ArkBotRequestBody{Input: ArkInput{Messages: msg}}
		arkResp := &ArkBotResponseBody{}
		base := strings.TrimRight(gpt.ArkApiUrl, "/")
		if !strings.Contains(base, "/bots") {
			base = base + "/bots"
		}
		endpoint := fmt.Sprintf("%s/%s/chat/completions", base, gpt.ArkBotId)
		err = gpt.sendRequestWithBodyType(endpoint, "POST", jsonBody, requestBody, arkResp)
		if err == nil && len(arkResp.Output.Choices) > 0 {
			return arkResp.Output.Choices[0].Message, nil
		}
		return Messages{}, errors.New("ark bots 请求失败")
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
