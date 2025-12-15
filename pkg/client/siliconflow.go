package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"aiblog/pkg/config"
)

type SiliconFlowClient struct {
	config *config.SiliconFlowConfig
	client *http.Client
}

type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	TopP        float64   `json:"top_p"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
}

type Choice struct {
	Index   int     `json:"index"`
	Message Message `json:"message"`
	Finish  string  `json:"finish_reason"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func NewSiliconFlowClient(config *config.SiliconFlowConfig) *SiliconFlowClient {
	return &SiliconFlowClient{
		config: config,
		client: &http.Client{
			Timeout: 6000 * time.Second,
		},
	}
}

func (c *SiliconFlowClient) GenerateDialogue(topic string) (string, error) {
	prompt := fmt.Sprintf(`作为一名资深的内容创作者，请为主题"%s"创作一段深度的对话播客内容。

要求：
1. 使用[S1]标记主持人，[S2]标记嘉宾
2. 每个标记占一行，内容另起一行
3. 对话要有深度和互动性
4. 总300-400长度字

格式示例：
[S1]
主持人说话内容...
[S2]
嘉宾说话内容...

对话内容要求：
- 开场引入：主持人介绍主题重要性
- 核心探讨：嘉宾分享专业见解和案例
- 深度挖掘：讨论挑战、机遇和未来趋势
- 总结升华：提炼要点，提供启发

请直接输出对话内容，不要包含其他说明文字。`, topic)

	req := ChatRequest{
		Model: c.config.ChatModel,
		Messages: []Message{
			{
				Role:    "system",
				Content: "你是一位专业的播客内容创作者，擅长创作深度、有趣的对话内容。生成的对话要格式清晰，每个说话者标记占一行。",
			},
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream:      false,
		Temperature: 0.8,
		MaxTokens:   2000,
		TopP:        0.9,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", c.config.BaseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("no choices returned from API")
	}

	return chatResp.Choices[0].Message.Content, nil
}

func (c *SiliconFlowClient) GenerateMultipleTopics(topics []string) ([]string, error) {
	dialogues := make([]string, 0, len(topics))

	for _, topic := range topics {
		dialogue, err := c.GenerateDialogue(topic)
		if err != nil {
			return nil, fmt.Errorf("failed to generate dialogue for topic '%s': %w", topic, err)
		}
		dialogues = append(dialogues, dialogue)

		// 添加延迟避免API限制
		time.Sleep(1 * time.Second)
	}

	return dialogues, nil
}
