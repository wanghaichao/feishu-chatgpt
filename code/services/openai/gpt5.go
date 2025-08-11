package openai

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

// 新增字段到你 ChatGPT 结构体里：
// AssistantID string
// mu          sync.Mutex

// Completions 基于 GPT-5 Assistants API 实现联网对话
func (gpt *ChatGPT) Completions(msg []Messages) (Messages, error) {
	gpt.mu.Lock()
	defer gpt.mu.Unlock()

	// 1. 如果没有 assistant_id，则创建一个
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

	// 2. 创建对话线程
	threadID, err := gpt.createThread()
	if err != nil {
		return Messages{}, err
	}

	// 3. 发送消息
	for _, m := range msg {
		if err := gpt.addMessage(threadID, m.Content); err != nil {
			return Messages{}, err
		}
	}

	// 4. 运行 assistant
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

// 辅助函数定义（使用你原有 sendRequestWithBodyType 函数）

func (gpt *ChatGPT) createAssistant() (string, error) {
	payload := map[string]interface{}{
		"name":         "Web-Enabled GPT-5",
		"model":        "gpt-5",
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
				Role    string            `json:"role"`
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

// 读取 assistant_id.txt 文件，缓存 assistant id
func (gpt *ChatGPT) loadAssistantID() (string, error) {
	data, err := ioutil.ReadFile("assistant_id.txt")
	if err != nil {
		return "", nil
	}
	return string(data), nil
}
