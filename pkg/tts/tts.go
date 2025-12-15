package tts

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

type TTSClient struct {
	config *config.TTSConfig
	client *http.Client
}

type TTSRequest struct {
	Model         string      `json:"model"`
	Input         string      `json:"input"`
	MaxTokens     int         `json:"max_tokens"`
	Voice         string      `json:"voice"`
	ResponseFormat string     `json:"response_format"`
	SampleRate    int         `json:"sample_rate"`
	Stream        bool        `json:"stream"`
	Speed         float64     `json:"speed"`
	Gain          float64     `json:"gain"`
	References    []Reference `json:"references,omitempty"`
}

type Reference struct {
	Audio string `json:"audio"`
	Text  string `json:"text"`
}

type AudioSegment struct {
	Speaker string
	Text    string
	Audio   []byte
}

func NewTTSClient(config *config.TTSConfig) *TTSClient {
	return &TTSClient{
		config: config,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (t *TTSClient) ParseDialogue(text string) ([]AudioSegment, error) {
	// 使用简单的字符串解析匹配 [S1] 和 [S2] 标记
	lines := strings.Split(text, "\n")
	segments := make([]AudioSegment, 0)
	var currentSpeaker string
	var currentText strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// 检查是否是说话者标记
		if strings.HasPrefix(line, "[S1]") {
			// 保存之前的内容
			if currentSpeaker != "" && currentText.Len() > 0 {
				content := strings.TrimSpace(currentText.String())
				if content != "" {
					segments = append(segments, AudioSegment{
						Speaker: currentSpeaker,
						Text:    content,
					})
				}
			}
			currentSpeaker = "[S1]"
			currentText.Reset()
			currentText.WriteString(strings.TrimSpace(line[4:]))
		} else if strings.HasPrefix(line, "[S2]") {
			// 保存之前的内容
			if currentSpeaker != "" && currentText.Len() > 0 {
				content := strings.TrimSpace(currentText.String())
				if content != "" {
					segments = append(segments, AudioSegment{
						Speaker: currentSpeaker,
						Text:    content,
					})
				}
			}
			currentSpeaker = "[S2]"
			currentText.Reset()
			currentText.WriteString(strings.TrimSpace(line[4:]))
		} else {
			// 继续当前说话者的内容
			if currentSpeaker != "" {
				currentText.WriteString(" ")
				currentText.WriteString(line)
			}
		}
	}

	// 保存最后的内容
	if currentSpeaker != "" && currentText.Len() > 0 {
		content := strings.TrimSpace(currentText.String())
		if content != "" {
			segments = append(segments, AudioSegment{
				Speaker: currentSpeaker,
				Text:    content,
			})
		}
	}

	if len(segments) == 0 {
		return nil, fmt.Errorf("no dialogue content parsed")
	}

	return segments, nil
}

func (t *TTSClient) GenerateAudio(segment AudioSegment, apiKey, baseURL, ttsModel string) ([]byte, error) {
	var voice string
	switch segment.Speaker {
	case "[S1]":
		voice = t.config.Voice1
	case "[S2]":
		voice = t.config.Voice2
	default:
		return nil, fmt.Errorf("unknown speaker: %s", segment.Speaker)
	}

	req := TTSRequest{
		Model:         ttsModel,
		Input:         segment.Text,
		MaxTokens:     t.config.MaxTokens,
		Voice:         voice,
		ResponseFormat: t.config.Format,
		SampleRate:    t.config.SampleRate,
		Stream:        false,
		Speed:         t.config.Speed,
		Gain:          t.config.Gain,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TTS request: %w", err)
	}

	httpReq, err := http.NewRequest("POST", baseURL+"/audio/speech", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create TTS request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	resp, err := t.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send TTS request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("TTS API request failed with status %d: %s", resp.StatusCode, string(body))
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read audio data: %w", err)
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("received empty audio data")
	}

	return audioData, nil
}

func (t *TTSClient) GenerateAllAudio(segments []AudioSegment, apiKey, baseURL, ttsModel string) ([]AudioSegment, error) {
	result := make([]AudioSegment, len(segments))

	for i, segment := range segments {
		fmt.Printf("正在生成第%d/%d段音频...\n", i+1, len(segments))

		audioData, err := t.GenerateAudio(segment, apiKey, baseURL, ttsModel)
		if err != nil {
			return nil, fmt.Errorf("failed to generate audio for segment %d: %w", i, err)
		}

		result[i] = AudioSegment{
			Speaker: segment.Speaker,
			Text:    segment.Text,
			Audio:   audioData,
		}

		// 添加短暂延迟避免API限制
		if i < len(segments)-1 {
			time.Sleep(500 * time.Millisecond)
		}
	}

	return result, nil
}

func (t *TTSClient) SaveAudioFiles(segments []AudioSegment, outputDir string) ([]string, error) {
	var filePaths []string

	for i, segment := range segments {
		speakerTag := strings.TrimPrefix(segment.Speaker, "[")
		speakerTag = strings.TrimSuffix(speakerTag, "]")

		filename := fmt.Sprintf("segment_%02d_S%s.%s", i+1, speakerTag, t.config.Format)

		filePath := filepath.Join(outputDir, filename)

		if err := os.WriteFile(filePath, segment.Audio, 0644); err != nil {
			return nil, fmt.Errorf("failed to save audio file %s: %w", filePath, err)
		}

		filePaths = append(filePaths, filePath)
	}

	return filePaths, nil
}