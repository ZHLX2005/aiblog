package composer

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"aiblog/pkg/config"
	"aiblog/pkg/ffmpeg"
	"aiblog/pkg/mapping"
)

type VideoComposer struct {
	ffmpeg   *ffmpeg.FFmpegProcessor
	config   *config.VideoConfig
}

type MediaPair struct {
	Speaker string // "S1" 或 "S2"
	Audio   string // 音频文件路径
	Image   string // 图片文件路径
}

type CompositionRule struct {
	Speaker1Image string // S1使用的图片模式（如 "p1.jpg"）
	Speaker2Image string // S2使用的图片模式（如 "p2.jpg"）
}

func NewVideoComposer(videoConfig *config.VideoConfig, ffmpegConfig *config.FFmpegConfig) *VideoComposer {
	return &VideoComposer{
		ffmpeg: ffmpeg.NewFFmpegProcessor(ffmpegConfig),
		config: videoConfig,
	}
}

// ComposeFromTOML 从TOML映射文件合成视频
func (vc *VideoComposer) ComposeFromTOML(tomlPath string) error {
	fmt.Printf("\n=== TOML映射文件合成模式 ===\n")
	fmt.Printf("映射文件: %s\n", tomlPath)

	// 加载映射文件
	mapping, err := mapping.LoadMappingFile(tomlPath)
	if err != nil {
		return fmt.Errorf("failed to load mapping file: %w", err)
	}

	// 验证映射文件
	if err := mapping.Validate(); err != nil {
		return fmt.Errorf("invalid mapping file: %w", err)
	}

	fmt.Printf("标题: %s\n", mapping.Title)
	fmt.Printf("音频数量: %d\n", mapping.AudioCount)
	fmt.Printf("描述: %s\n", mapping.Description)

	// 创建配对列表
	pairs := make([]MediaPair, 0, len(mapping.AudioFiles))
	for i, audio := range mapping.AudioFiles {
		fmt.Printf("\n音频 %d/%d:\n", i+1, len(mapping.AudioFiles))
		fmt.Printf("  文件: %s\n", filepath.Base(audio.AudioFile))
		fmt.Printf("  说话者: %s\n", audio.Speaker)
		fmt.Printf("  内容: %s\n", audio.Content)
		fmt.Printf("  图片: %s\n", audio.ImageFile)

		pairs = append(pairs, MediaPair{
			Speaker: audio.Speaker,
			Audio:   audio.AudioFile,
			Image:   audio.ImageFile,
		})
	}

	// 使用映射文件中的输出路径或默认路径
	outputPath := mapping.OutputPath
	if outputPath == "" || outputPath == "./composed_video.mp4" {
		// 生成基于标题的输出文件名
		title := strings.ReplaceAll(mapping.Title, " ", "_")
		title = strings.ReplaceAll(title, "：", "_")
		title = strings.ReplaceAll(title, "，", "_")
		title = strings.ReplaceAll(title, "。", "_")
		outputPath = fmt.Sprintf("%s_composed.mp4", title)
	}

	return vc.ComposeVideoFromPairs(pairs, outputPath)
}

// ComposeVideoFromPairs 根据音频图片配对合成视频
func (vc *VideoComposer) ComposeVideoFromPairs(pairs []MediaPair, outputPath string) error {
	if len(pairs) == 0 {
		return fmt.Errorf("no media pairs provided")
	}

	fmt.Printf("\n=== 视频合成工具 ===\n")
	fmt.Printf("配对数量: %d\n", len(pairs))
	fmt.Printf("输出路径: %s\n", outputPath)

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "video_compose_")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 为每个配对生成视频片段
	videoSegments := make([]string, len(pairs))
	for i, pair := range pairs {
		fmt.Printf("\n处理配对 %d/%d:\n", i+1, len(pairs))
		fmt.Printf("  说话者: %s\n", pair.Speaker)
		fmt.Printf("  音频: %s\n", filepath.Base(pair.Audio))
		fmt.Printf("  图片: %s\n", filepath.Base(pair.Image))

		// 验证文件存在
		if _, err := os.Stat(pair.Audio); os.IsNotExist(err) {
			return fmt.Errorf("audio file not found: %s", pair.Audio)
		}
		if _, err := os.Stat(pair.Image); os.IsNotExist(err) {
			return fmt.Errorf("image file not found: %s", pair.Image)
		}

		// 生成单个视频片段
		segmentPath := filepath.Join(tempDir, fmt.Sprintf("segment_%03d.mp4", i))
		if err := vc.ffmpeg.GenerateVideoSegment(pair.Image, pair.Audio, segmentPath, vc.config); err != nil {
			return fmt.Errorf("failed to create video segment %d: %w", i, err)
		}

		videoSegments[i] = segmentPath
		fmt.Printf("  ✓ 片段生成完成\n")
	}

	// 合并所有视频片段
	fmt.Printf("\n合并视频片段...\n")
	if err := vc.ffmpeg.MergeVideoSegments(videoSegments, outputPath); err != nil {
		return fmt.Errorf("failed to merge video segments: %w", err)
	}

	fmt.Printf("\n✅ 视频合成完成: %s\n", outputPath)
	return nil
}

// ComposeFromDirectories 从目录中自动匹配音频和图片文件
func (vc *VideoComposer) ComposeFromDirectories(audioDir, imageDir, outputPath string, rule *CompositionRule) error {
	fmt.Printf("\n=== 目录自动匹配模式 ===\n")
	fmt.Printf("音频目录: %s\n", audioDir)
	fmt.Printf("图片目录: %s\n", imageDir)

	// 获取所有音频文件
	audioFiles, err := filepath.Glob(filepath.Join(audioDir, "*.mp3"))
	if err != nil {
		return fmt.Errorf("failed to find audio files: %w", err)
	}

	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files found in directory: %s", audioDir)
	}

	// 获取所有图片文件
	imageFiles, err := vc.findImageFiles(imageDir)
	if err != nil {
		return fmt.Errorf("failed to find image files: %w", err)
	}

	fmt.Printf("找到音频文件: %d个\n", len(audioFiles))
	fmt.Printf("找到图片文件: %d个\n", len(imageFiles))

	// 匹配规则
	if rule == nil {
		// 默认规则：p1.jpg对应S1，p2.jpg对应S2
		rule = &CompositionRule{
			Speaker1Image: "p1",
			Speaker2Image: "p2",
		}
	}

	// 按文件名排序以确保一致性
	sort.Strings(audioFiles)
	sort.Strings(imageFiles)

	// 创建配对
	pairs := make([]MediaPair, 0)
	s1Images := vc.filterImagesByName(imageFiles, rule.Speaker1Image)
	s2Images := vc.filterImagesByName(imageFiles, rule.Speaker2Image)

	for i, audioFile := range audioFiles {
		// 从文件名判断说话者
		speaker := vc.detectSpeakerFromFilename(audioFile)
		var imageFile string

		if speaker == "S1" && len(s1Images) > 0 {
			imageFile = s1Images[i%len(s1Images)]
		} else if speaker == "S2" && len(s2Images) > 0 {
			imageFile = s2Images[i%len(s2Images)]
		} else {
			// 默认使用第一张图片
			if len(imageFiles) > 0 {
				imageFile = imageFiles[i%len(imageFiles)]
			} else {
				return fmt.Errorf("no matching image found for audio: %s", audioFile)
			}
		}

		pairs = append(pairs, MediaPair{
			Speaker: speaker,
			Audio:   audioFile,
			Image:   imageFile,
		})
	}

	return vc.ComposeVideoFromPairs(pairs, outputPath)
}

// findImageFiles 查找所有支持的图片文件
func (vc *VideoComposer) findImageFiles(dir string) ([]string, error) {
	var imageFiles []string

	for _, pattern := range vc.config.ImagePatterns {
		files, err := filepath.Glob(filepath.Join(dir, pattern))
		if err != nil {
			return nil, err
		}
		imageFiles = append(imageFiles, files...)
	}

	return imageFiles, nil
}

// filterImagesByName 根据名称模式过滤图片
func (vc *VideoComposer) filterImagesByName(images []string, pattern string) []string {
	var filtered []string
	for _, img := range images {
		if strings.Contains(strings.ToLower(filepath.Base(img)), strings.ToLower(pattern)) {
			filtered = append(filtered, img)
		}
	}
	return filtered
}

// detectSpeakerFromFilename 从文件名检测说话者
func (vc *VideoComposer) detectSpeakerFromFilename(filename string) string {
	base := strings.ToLower(filepath.Base(filename))
	if strings.Contains(base, "s1") || strings.Contains(base, "speaker1") {
		return "S1"
	}
	if strings.Contains(base, "s2") || strings.Contains(base, "speaker2") {
		return "S2"
	}
	// 默认返回S1
	return "S1"
}

// ComposeWithCustomPattern 使用自定义模式匹配图片和音频
func (vc *VideoComposer) ComposeWithCustomPattern(audioDir, imageDir, outputPath string, pattern string) error {
	fmt.Printf("\n=== 自定义模式匹配 ===\n")
	fmt.Printf("匹配模式: %s\n", pattern)

	// 解析模式（例如：p1,p2,p1,p2 表示交替使用p1和p2）
	patterns := strings.Split(pattern, ",")
	if len(patterns) == 0 {
		return fmt.Errorf("invalid pattern")
	}

	// 获取所有音频文件
	audioFiles, err := filepath.Glob(filepath.Join(audioDir, "*.mp3"))
	if err != nil {
		return fmt.Errorf("failed to find audio files: %w", err)
	}

	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files found")
	}

	// 获取所有图片文件
	imageFiles, err := vc.findImageFiles(imageDir)
	if err != nil {
		return fmt.Errorf("failed to find image files: %w", err)
	}

	sort.Strings(audioFiles)
	sort.Strings(imageFiles)

	// 创建配对
	pairs := make([]MediaPair, 0, len(audioFiles))
	for i, audioFile := range audioFiles {
		// 获取当前模式
		currentPattern := patterns[i%len(patterns)]
		currentPattern = strings.TrimSpace(currentPattern)

		// 查找匹配的图片
		var selectedImage string
		for _, img := range imageFiles {
			if strings.Contains(strings.ToLower(filepath.Base(img)), strings.ToLower(currentPattern)) {
				selectedImage = img
				break
			}
		}

		if selectedImage == "" {
			return fmt.Errorf("no matching image found for pattern: %s", currentPattern)
		}

		pairs = append(pairs, MediaPair{
			Speaker: vc.detectSpeakerFromFilename(audioFile),
			Audio:   audioFile,
			Image:   selectedImage,
		})

		fmt.Printf("配对 %d: %s -> %s\n", i+1, filepath.Base(audioFile), filepath.Base(selectedImage))
	}

	return vc.ComposeVideoFromPairs(pairs, outputPath)
}

// BatchCompose 批量合成多个视频
func (vc *VideoComposer) BatchCompose(audioDir, imageDir, outputDir string, batchSize int, pattern string) error {
	// 获取所有音频文件
	audioFiles, err := filepath.Glob(filepath.Join(audioDir, "*.mp3"))
	if err != nil {
		return fmt.Errorf("failed to find audio files: %w", err)
	}

	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files found")
	}

	sort.Strings(audioFiles)

	// 分批处理
	for i := 0; i < len(audioFiles); i += batchSize {
		end := i + batchSize
		if end > len(audioFiles) {
			end = len(audioFiles)
		}

		batchAudio := audioFiles[i:end]
		batchName := fmt.Sprintf("batch_%03d", i/batchSize+1)

		// 创建批次目录
		batchAudioDir := filepath.Join(outputDir, "temp", batchName, "audio")
		batchImageDir := filepath.Join(outputDir, "temp", batchName, "images")
		os.MkdirAll(batchAudioDir, 0755)
		os.MkdirAll(batchImageDir, 0755)

		// 复制音频文件
		for j, audioFile := range batchAudio {
			newPath := filepath.Join(batchAudioDir, fmt.Sprintf("%03d_%s", j, filepath.Base(audioFile)))
			copyFile(audioFile, newPath)
		}

		// 生成输出路径
		outputPath := filepath.Join(outputDir, fmt.Sprintf("%s.mp4", batchName))

		// 合成视频
		if err := vc.ComposeWithCustomPattern(batchAudioDir, batchImageDir, outputPath, pattern); err != nil {
			fmt.Printf("批次 %s 失败: %v\n", batchName, err)
			continue
		}

		fmt.Printf("批次 %s 完成\n", batchName)
	}

	return nil
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = destination.ReadFrom(source)
	return err
}