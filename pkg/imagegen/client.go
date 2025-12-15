package imagegen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiblog/pkg/config"
)

type ImageGenClient struct {
	config *config.SiliconFlowConfig
	client *http.Client
}

type ImageRequest struct {
	Model              string   `json:"model"`
	Prompt             string   `json:"prompt"`
	NegativePrompt     string   `json:"negative_prompt,omitempty"`
	ImageSize          string   `json:"image_size"`
	BatchSize          int      `json:"batch_size,omitempty"`
	Seed               int64    `json:"seed,omitempty"`
	NumInferenceSteps  int      `json:"num_inference_steps,omitempty"`
	GuidanceScale      float64  `json:"guidance_scale,omitempty"`
	Cfg                float64  `json:"cfg,omitempty"`
	Image              string   `json:"image,omitempty"`
	Image2             string   `json:"image2,omitempty"`
	Image3             string   `json:"image3,omitempty"`
}

type ImageResponse struct {
	Images  []ImageItem `json:"images"`
	Timings Timings     `json:"timings"`
	Seed    int         `json:"seed"`
}

type ImageItem struct {
	URL      string `json:"url"`
	Width    int    `json:"width"`
	Height   int    `json:"height"`
	Content  string `json:"content,omitempty"`
}

type Timings struct {
	Inference float64 `json:"inference"`
}

func NewImageGenClient(config *config.SiliconFlowConfig) *ImageGenClient {
	return &ImageGenClient{
		config: config,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *ImageGenClient) GenerateImageForSegment(segmentText, topic string, outputDir string) (string, error) {
	// 构建适合PPT的提示词
	prompt := c.buildPromptForSegment(segmentText, topic)

	// 创建图片生成请求
	req := ImageRequest{
		Model:             "Qwen/Qwen-Image",
		Prompt:            prompt,
		ImageSize:         "1664x928", // 16:9 适合视频
		BatchSize:         1,
		NumInferenceSteps: 20,
		Cfg:               4.0,
		Seed:              time.Now().UnixNano() % 10000000000,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("failed to marshal image request: %w", err)
	}

	// 发送请求
	httpReq, err := http.NewRequest("POST", c.config.BaseURL+"/images/generations", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create image request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("failed to send image request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("image API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var imgResp ImageResponse
	if err := json.NewDecoder(resp.Body).Decode(&imgResp); err != nil {
		return "", fmt.Errorf("failed to decode image response: %w", err)
	}

	if len(imgResp.Images) == 0 {
		return "", fmt.Errorf("no images returned from API")
	}

	// 下载图片
	imageURL := imgResp.Images[0].URL
	imagePath, err := c.downloadImage(imageURL, outputDir, segmentText)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}

	return imagePath, nil
}

func (c *ImageGenClient) buildPromptForSegment(segmentText, topic string) string {
	// 提取关键内容
	keywords := c.extractKeywords(segmentText)

	// 构建PPT风格的提示词
	prompt := fmt.Sprintf(`Create a professional PPT-style infographic or diagram about: %s

Content context: %s

Requirements:
- Clean, professional presentation style
- Business or educational infographic design
- Clear visual hierarchy
- Minimalist background
- High contrast for readability
- Modern color scheme
- Include relevant icons or simple illustrations
- Suitable for business presentation
- No text overlay, just visual content

Style: corporate presentation, clean design, infographic, educational visual`,
		topic,
		keywords)

	return prompt
}

func (c *ImageGenClient) extractKeywords(text string) string {
	// 简单的关键词提取
	words := strings.Fields(strings.ToLower(text))
	var keywords []string

	// 过滤停用词并提取重要词汇
	stopWords := map[string]bool{
		"the": true, "a": true, "an": true, "and": true, "or": true, "but": true,
		"in": true, "on": true, "at": true, "to": true, "for": true, "of": true,
		"with": true, "by": true, "is": true, "are": true, "was": true, "were": true,
		"的": true, "了": true, "在": true, "是": true, "有": true,
		"和": true, "就": true, "不": true, "人": true, "都": true, "一": true,
		"个": true, "上": true, "也": true, "很": true, "到": true, "说": true,
	}

	for _, word := range words {
		if len(word) > 2 && !stopWords[word] {
			keywords = append(keywords, word)
			if len(keywords) >= 10 {
				break
			}
		}
	}

	return strings.Join(keywords, ", ")
}

func (c *ImageGenClient) downloadImage(imageURL, outputDir string, segmentText string) (string, error) {
	// 发送HTTP请求下载图片
	resp, err := c.client.Get(imageURL)
	if err != nil {
		return "", fmt.Errorf("failed to download image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("image download failed with status %d", resp.StatusCode)
	}

	// 从URL获取文件扩展名
	ext := ".jpg"
	if strings.Contains(imageURL, ".png") {
		ext = ".png"
	}

	// 生成文件名
	timestamp := time.Now().Format("20060102_150405")
	// 使用段落的简短摘要作为文件名的一部分
	words := strings.Fields(segmentText)
	if len(words) == 0 {
		words = []string{"image"}
	}

	maxWords := 5
	if len(words) < maxWords {
		maxWords = len(words)
	}

	preview := strings.Join(words[:maxWords], "_")
	preview = strings.ReplaceAll(preview, ",", "")
	preview = strings.ReplaceAll(preview, "。", "")
	preview = strings.ReplaceAll(preview, "？", "")
	preview = strings.ReplaceAll(preview, "！", "")

	if len(preview) > 50 {
		preview = preview[:50]
	}

	filename := fmt.Sprintf("ppt_%s_%s%s", timestamp, preview, ext)
	filePath := filepath.Join(outputDir, filename)

	// 保存图片
	file, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create image file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to save image: %w", err)
	}

	return filePath, nil
}

func (c *ImageGenClient) GenerateImagesForSegments(segments []TextSegment, topic string, outputDir string) ([]string, error) {
	var imagePaths []string

	for i, segment := range segments {
		fmt.Printf("正在生成第%d/%d张图片...\n", i+1, len(segments))

		imagePath, err := c.GenerateImageForSegment(segment.Text, topic, outputDir)
		if err != nil {
			fmt.Printf("生成图片失败(段落%d): %v\n", i+1, err)
			continue
		}

		imagePaths = append(imagePaths, imagePath)

		// 添加延迟避免API限制
		if i < len(segments)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return imagePaths, nil
}

// TextSegment 表示文本段落
type TextSegment struct {
	Index   int
	Text    string
	Speaker string
}