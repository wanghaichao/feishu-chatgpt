package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

// 给 ChatGPT 结构体新增字段：
func (gpt *ChatGPT) ensureFields() {
	// 反射或初始化不能改现有结构体，这里用sync.Once做初始化保护的替代，简化用mutex
	// 如果你需要，可在ChatGPT定义添加下面字段：
	// AssistantID string
	// mu sync.Mutex
}

// 这里假设你已在ChatGPT结构体加了下面字段（如果没加，需要你加）：
/*
type ChatGPT struct {
	// 你原有字段...
	AssistantID string
	mu          sync.Mutex
}
*/

// Completions 使用 Assistants API 与浏览器工具
func (gpt *ChatGPT) Completions(msg []Messages) (Messages, error) {
	gpt.mu.Lock()
	defer gpt.mu.Unlock()

	// 1. 读取缓存或创建 Assistant
	if gpt.AssistantID == "" {
		id, err := gpt.loadAssistantID()
		if err != nil || id == "" {
			id, err = gpt.createAssistant()
			if err != nil {
				return Messages{}, err
			}
			_ = ioutil.WriteFile("assistant_id.txt", []byte(id), 0644)
		}
		gpt.AssistantID = id
	}

	// 2. 创建线程
	threadID, err := gpt.createThread()
	if err != nil {
		return Messages{}, err
	}

	// 3. 发送用户消息
	for _, m := range msg {
		if err := gpt.addMessage(threadID, m.Content); err != nil {
			return Messages{}, err
		}
	}

	// 4. 运行 Assistant
	if err := gpt.runAssistant(threadID, gpt.AssistantID); err != nil {
		return Messages{}, err
	}

	// 5. 获取回答
	outputs, err := gpt.getMessages(threadID)
	if err != nil {
		return Messages{}, err
	}
	if len(outputs) == 0 {
		return Messages{}, errors.New("AI 没有返回内容")
	}

	return Messages{Role: "assistant", Content: outputs[0]}, nil
}

// 下面是辅助方法，所有请求都走你现有 sendRequestWithBodyType，且兼容多 API Key 负载均衡

func (gpt *ChatGPT) createAssistant() (string, error) {
	payload := map[string]interface{}{
		"name":         "Web-Enabled GPT-5",
		"model":        "gpt-5",
		"tools":        []map[string]string{{"type": "browser"}},
		"instructions": "You are a helpful assistant with web browsing capabilities.",
	}
	var result struct {
		ID string `json:"id"`
	}
	err := gpt.sendRequestWithBodyType(gpt.ApiUrl+"/assistants", "POST", jsonBody, payload, &result)
	if err != nil {
		return "", fmt.Errorf("创建 Assistant 失败: %w", err)
	}
	return result.ID, nil
}

func (gpt *ChatGPT) createThread() (string, error) {
	var result struct {
		ID string `json:"id"`
	}
	err := gpt.sendRequestWithBodyType(gpt.ApiUrl+"/threads", "POST", jsonBody, nil, &result)
	if err != nil {
		return "", fmt.Errorf("创建 Thread 失败: %w", err)
	}
	return result.ID, nil
}

func (gpt *ChatGPT) addMessage(threadID, content string) error {
	payload := map[string]interface{}{
		"role":    "user",
		"content": content,
	}
	return gpt.sendRequestWithBodyType(fmt.Sprintf("%s/threads/%s/messages", gpt.ApiUrl, threadID), "POST", jsonBody, payload, nil)
}

func (gpt *ChatGPT) runAssistant(threadID, assistantID string) error {
	payload := map[string]interface{}{
		"assistant_id": assistantID,
	}
	return gpt.sendRequestWithBodyType(fmt.Sprintf("%s/threads/%s/runs", gpt.ApiUrl, threadID), "POST", jsonBody, payload, nil)
}

func (gpt *ChatGPT) getMessages(threadID string) ([]string, error) {
	reqUrl := fmt.Sprintf("%s/threads/%s/messages", gpt.ApiUrl, threadID)
	client := &http.Client{Timeout: 20 * time.Second}

	for i := 0; i < 10; i++ {
		req, err := http.NewRequest("GET", reqUrl, nil)
		if err != nil {
			return nil, err
		}
		// 走负载均衡拿 ApiKey
		api := gpt.Lb.GetAPI()
		req.Header.Set("Authorization", "Bearer "+api.Key)

		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var result struct {
			Data []struct {
				Role    string          `json:"role"`
				Content []json.RawMessage `json:"content"`
			} `json:"data"`
		}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		var outputs []string
		for _, d := range result.Data {
			if d.Role == "assistant" {
				for _, c := range d.Content {
					var cm map[string]interface{}
					if err := json.Unmarshal(c, &cm); err == nil {
						if cm["type"] == "text" {
							if textVal, ok := cm["text"].(map[string]interface{}); ok {
								if val, ok := textVal["value"].(string); ok {
									outputs = append(outputs, val)
								}
							}
						}
					}
				}
			}
		}
		if len(outputs) > 0 {
			return outputs, nil
		}
		time.Sleep(2 * time.Second)
	}
	return nil, errors.New("等待超时，无返回消息")
}

// 读取 AssistantID 缓存
func (gpt *ChatGPT) loadAssistantID() (string, error) {
	data, err := ioutil.ReadFile("assistant_id.txt")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}
	return string(data), nil
}
