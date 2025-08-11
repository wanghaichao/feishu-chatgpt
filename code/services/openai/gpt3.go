package openai

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync"
	"time"
)

const engine = "gpt-5"
const cacheFile = "assistant_id.txt"

type Messages struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatGPT struct {
	ApiUrl      string
	ApiKey      string
	AssistantID string
	mu          sync.Mutex // 保证多协程安全
}

// NewChatGPT 从环境变量读取配置初始化
func NewChatGPT() *ChatGPT {
	apiKey := os.Getenv("OPENAI_KEY")
	apiUrl := os.Getenv("OPENAI_API_URL")
	if apiUrl == "" {
		apiUrl = "https://api.openai.com/v1"
	}
	return &ChatGPT{
		ApiKey: apiKey,
		ApiUrl: apiUrl,
	}
}

// Completions 保持旧接口，内部完成联网GPT-5 Assistants API调用，带浏览器联网能力
func (gpt *ChatGPT) Completions(msg []Messages) (Messages, error) {
	gpt.mu.Lock()
	defer gpt.mu.Unlock()

	// 1. 加载缓存或创建 Assistant
	if gpt.AssistantID == "" {
		id, err := gpt.loadAssistantID()
		if err != nil || id == "" {
			id, err = gpt.createAssistant()
			if err != nil {
				return Messages{}, err
			}
			_ = ioutil.WriteFile(cacheFile, []byte(id), 0644)
		}
		gpt.AssistantID = id
	}

	// 2. 创建 Thread
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

	// 5. 获取返回内容
	outputs, err := gpt.getMessages(threadID)
	if err != nil {
		return Messages{}, err
	}
	if len(outputs) == 0 {
		return Messages{}, errors.New("AI 没有返回内容")
	}

	return Messages{
		Role:    "assistant",
		Content: outputs[0],
	}, nil
}

// ---- Assistants API ----

func (gpt *ChatGPT) createAssistant() (string, error) {
	payload := map[string]interface{}{
		"name":         "Web-Enabled GPT-5",
		"model":        engine,
		"tools":        []map[string]string{{"type": "browser"}},
		"instructions": "You are a helpful assistant with web browsing capabilities.",
	}
	return gpt.postAndGetID("/assistants", payload, "创建 Assistant 失败")
}

func (gpt *ChatGPT) createThread() (string, error) {
	return gpt.postAndGetID("/threads", nil, "创建 Thread 失败")
}

func (gpt *ChatGPT) addMessage(threadID, content string) error {
	payload := map[string]interface{}{
		"role":    "user",
		"content": content,
	}
	return gpt.postNoReturn(fmt.Sprintf("/threads/%s/messages", threadID), payload, "添加消息失败")
}

func (gpt *ChatGPT) runAssistant(threadID, assistantID string) error {
	payload := map[string]interface{}{
		"assistant_id": assistantID,
	}
	return gpt.postNoReturn(fmt.Sprintf("/threads/%s/runs", threadID), payload, "运行 Assistant 失败")
}

func (gpt *ChatGPT) getMessages(threadID string) ([]string, error) {
	req, _ := http.NewRequest("GET", gpt.ApiUrl+"/threads/"+threadID+"/messages", nil)
	req.Header.Set("Authorization", "Bearer "+gpt.ApiKey)

	for i := 0; i < 10; i++ {
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		body, _ := ioutil.ReadAll(resp.Body)
		resp.Body.Close()

		var result map[string]interface{}
		if err := json.Unmarshal(body, &result); err != nil {
			return nil, err
		}

		if data, ok := result["data"].([]interface{}); ok {
			var outputs []string
			for _, msg := range data {
				m := msg.(map[string]interface{})
				if m["role"] == "assistant" {
					if content, ok := m["content"].([]interface{}); ok {
						for _, c := range content {
							cm := c.(map[string]interface{})
							if cm["type"] == "text" {
								text := cm["text"].(map[string]interface{})
								outputs = append(outputs, text["value"].(string))
							}
						}
					}
				}
			}
			if len(outputs) > 0 {
				return outputs, nil
			}
		}

		time.Sleep(2 * time.Second)
	}
	return nil, errors.New("等待超时，无返回消息")
}

// ---- 缓存操作 ----
func (gpt *ChatGPT) loadAssistantID() (string, error) {
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return "", nil
	}
	data, err := ioutil.ReadFile(cacheFile)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ---- HTTP 请求辅助 ----
func (gpt *ChatGPT) postAndGetID(path string, payload interface{}, errMsg string) (string, error) {
	var buf *bytes.Buffer
	if payload != nil {
		data, _ := json.Marshal(payload)
		buf = bytes.NewBuffer(data)
	} else {
		buf = bytes.NewBuffer([]byte{})
	}

	req, _ := http.NewRequest("POST", gpt.ApiUrl+path, buf)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gpt.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	if id, ok := result["id"].(string); ok {
		return id, nil
	}
	return "", errors.New(errMsg + ": " + string(body))
}

func (gpt *ChatGPT) postNoReturn(path string, payload interface{}, errMsg string) error {
	data, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", gpt.ApiUrl+path, bytes.NewBuffer(data))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+gpt.ApiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		return errors.New(errMsg + ": " + string(body))
	}
	return nil
}
