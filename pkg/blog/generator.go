package blog

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiblog/pkg/client"
	"aiblog/pkg/config"
	"aiblog/pkg/ffmpeg"
	"aiblog/pkg/images"
	"aiblog/pkg/mapping"
	"aiblog/pkg/tts"
)

type BlogGenerator struct {
	config      *config.Config
	sfClient    *client.SiliconFlowClient
	ttsClient   *tts.TTSClient
	ffmpeg      *ffmpeg.FFmpegProcessor
	imageMgr    *images.ImageManager
	sessionDir  string // 当前会话目录
}

type BlogPost struct {
	Topic         string
	Dialogue      string
	AudioSegments []tts.AudioSegment
	AudioFiles    []string
	FinalAudio    string
	TextFile      string
	// 视频相关字段
	VideoFiles    []string
	FinalVideo    string
	GenerateVideo bool
	GeneratedImages []string // AI生成的图片路径
	SessionDir    string      // 会话目录
}

func NewBlogGenerator(cfg *config.Config) *BlogGenerator {
	return &BlogGenerator{
		config:    cfg,
		sfClient:  client.NewSiliconFlowClient(&cfg.SiliconFlow),
		ttsClient: tts.NewTTSClient(&cfg.TTS),
		ffmpeg:    ffmpeg.NewFFmpegProcessor(&cfg.FFmpeg),
		imageMgr:  images.NewImageManager(&cfg.Video, &cfg.SiliconFlow),
	}
}

func (bg *BlogGenerator) initSession() error {
	if !bg.config.Output.SessionFolders {
		return nil
	}

	// 创建新的会话目录
	sessionDir, err := bg.config.CreateSessionDir()
	if err != nil {
		return fmt.Errorf("failed to create session directory: %w", err)
	}
	bg.sessionDir = sessionDir

	// 在会话目录下创建子目录
	subDirs := []string{"audio", "video", "images", "text"}
	for _, dir := range subDirs {
		fullPath := filepath.Join(sessionDir, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	fmt.Printf("会话目录: %s\n", sessionDir)
	return nil
}

func (bg *BlogGenerator) GenerateSingleBlog(topic string, generateVideo bool, useAIImages bool) (*BlogPost, error) {
	fmt.Printf("\n=== AI博客生成器 ===\n")
	fmt.Printf("主题: %s\n", topic)
	if generateVideo {
		fmt.Printf("模式: 音频 + 视频")
		if useAIImages {
			fmt.Printf(" (AI生成图片)")
		} else {
			fmt.Printf(" (使用现有图片)")
		}
	} else {
		fmt.Printf("模式: 仅音频")
	}
	fmt.Println("\n-------------------")

	// 初始化会话
	if err := bg.initSession(); err != nil {
		return nil, fmt.Errorf("failed to initialize session: %w", err)
	}

	// 1. 生成深度对话内容
	fmt.Println("步骤1: 生成对话内容...")
	dialogue, err := bg.sfClient.GenerateDialogue(topic)
	if err != nil {
		return nil, fmt.Errorf("生成对话失败: %w", err)
	}
	fmt.Println("✓ 对话内容生成完成")

	// 保存对话文本
	timestamp := time.Now().Format("20060102_150405")
	sanitizedTopic := strings.ReplaceAll(strings.ToLower(topic), " ", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "：", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "，", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "。", "_")

	var textFile string
	if bg.sessionDir != "" {
		textFile = filepath.Join(bg.sessionDir, "text", fmt.Sprintf("%s_%s.txt", sanitizedTopic, timestamp))
	} else {
		textFile = filepath.Join(bg.config.Output.Dir, fmt.Sprintf("%s_%s.txt", sanitizedTopic, timestamp))
	}

	if err := os.WriteFile(textFile, []byte(dialogue), 0644); err != nil {
		return nil, fmt.Errorf("保存对话文本失败: %w", err)
	}
	fmt.Printf("✓ 文本已保存: %s\n", filepath.Base(textFile))

	// 创建音频映射文件
	mappingFile := mapping.NewMappingFile(topic, "AI生成的博客对话，请为每个音频文件指定对应的图片")

	// 2. 解析对话为音频片段
	fmt.Println("\n步骤2: 解析对话结构...")
	segments, err := bg.ttsClient.ParseDialogue(dialogue)
	if err != nil {
		return nil, fmt.Errorf("解析对话失败: %w", err)
	}
	fmt.Printf("✓ 解析完成，共%d个对话段落\n", len(segments))

	// 显示对话概览
	fmt.Println("\n对话概览:")
	for _, seg := range segments {
		preview := seg.Text
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		fmt.Printf("  [%s] %s\n", seg.Speaker, preview)
	}

	// 3. 生成音频
	fmt.Println("\n步骤3: 生成语音音频...")
	audioSegments, err := bg.ttsClient.GenerateAllAudio(
		segments,
		bg.config.SiliconFlow.APIKey,
		bg.config.SiliconFlow.BaseURL,
		bg.config.SiliconFlow.TTSModel,
	)
	if err != nil {
		return nil, fmt.Errorf("生成音频失败: %w", err)
	}
	fmt.Println("✓ 音频生成完成")

	// 4. 保存单个音频文件
	fmt.Println("\n步骤4: 保存音频文件...")
	audioFiles, err := bg.saveAudioFiles(audioSegments, mappingFile)
	if err != nil {
		return nil, fmt.Errorf("保存音频文件失败: %w", err)
	}
	fmt.Printf("✓ 已保存%d个音频文件\n", len(audioFiles))

	// 保存TOML映射文件
	var tomlPath string
	if bg.sessionDir != "" {
		tomlPath = filepath.Join(bg.sessionDir, fmt.Sprintf("%s_%s_mapping.toml", sanitizedTopic, timestamp))
	} else {
		tomlPath = filepath.Join(bg.config.Output.Dir, fmt.Sprintf("%s_%s_mapping.toml", sanitizedTopic, timestamp))
	}

	if err := mappingFile.SaveToFile(tomlPath); err != nil {
		fmt.Printf("⚠️ 保存映射文件失败: %v\n", err)
	} else {
		fmt.Printf("✓ 映射文件已保存: %s\n", filepath.Base(tomlPath))
		fmt.Printf("💡 提示: 可以编辑此文件指定图片路径，然后使用compose命令生成视频\n")
	}

	// 初始化博客对象
	blogPost := &BlogPost{
		Topic:         topic,
		Dialogue:      dialogue,
		AudioSegments: audioSegments,
		AudioFiles:    audioFiles,
		TextFile:      textFile,
		GenerateVideo: generateVideo,
		SessionDir:    bg.sessionDir,
	}

	// 5. 生成图片（如果需要视频）
	var imageFiles []string
	if generateVideo && bg.config.Video.Enabled {
		fmt.Println("\n步骤5: 处理图片...")
		var err error

		if useAIImages {
			fmt.Println("  使用AI生成PPT风格图片...")
			var imageDir string
			if bg.sessionDir != "" {
				imageDir = filepath.Join(bg.sessionDir, "images")
			} else {
				imageDir = bg.config.Output.VideoDir
			}
			imageFiles, err = bg.imageMgr.GetImagesForSegments(
				audioSegments,
				topic,
				true, // 使用AI生成
				imageDir,
			)
			if err != nil {
				fmt.Printf("⚠️ AI图片生成失败: %v\n", err)
				fmt.Println("尝试使用现有图片...")
				imageFiles, err = bg.imageMgr.GetImagesForSegments(
					audioSegments,
					topic,
					false, // 使用现有图片
					bg.config.Output.VideoDir,
				)
			} else {
				blogPost.GeneratedImages = imageFiles
			}
		} else {
			fmt.Println("  使用现有图片...")
			imageFiles, err = bg.imageMgr.GetImagesForSegments(
				audioSegments,
				topic,
				false, // 使用现有图片
				bg.config.Output.VideoDir,
			)
		}

		if err != nil {
			fmt.Printf("⚠️ 图片处理失败: %v\n", err)
			fmt.Println("继续完成音频生成...")
		} else {
			fmt.Printf("✓ 图片准备完成，共%d张\n", len(imageFiles))
		}
	}

	// 6. 合并音频文件
	fmt.Println("\n步骤6: 合成播客音频...")
	var finalAudioPath string
	if bg.sessionDir != "" {
		finalAudioPath = filepath.Join(bg.sessionDir, "video", fmt.Sprintf("podcast_%s_%s.%s", sanitizedTopic, timestamp, bg.config.FFmpeg.OutputFormat))
	} else {
		finalAudioPath = filepath.Join(bg.config.Output.Dir, fmt.Sprintf("podcast_%s_%s.%s", sanitizedTopic, timestamp, bg.config.FFmpeg.OutputFormat))
	}

	// 创建临时音频文件列表
	audioDir := filepath.Join(bg.sessionDir, "audio")
	tempAudioFiles := make([]string, len(audioFiles))
	for i, audioFile := range audioFiles {
		// 复制到音频目录（如果不在同一个目录）
		if audioDir != filepath.Dir(audioFile) {
			tempName := fmt.Sprintf("segment_%02d.%s", i+1, bg.config.TTS.Format)
			tempPath := filepath.Join(audioDir, tempName)
			if err := copyFile(audioFile, tempPath); err != nil {
				return nil, fmt.Errorf("failed to copy audio file: %w", err)
			}
			tempAudioFiles[i] = tempPath
		} else {
			tempAudioFiles[i] = audioFile
		}
	}

	if err := bg.ffmpeg.MergeAudioFiles(tempAudioFiles, finalAudioPath); err != nil {
		return nil, fmt.Errorf("生成播客失败: %w", err)
	}
	blogPost.FinalAudio = finalAudioPath
	fmt.Printf("✓ 播客生成完成: %s\n", filepath.Base(finalAudioPath))

	// 7. 生成视频（如果启用且有图片）
	if generateVideo && bg.config.Video.Enabled && len(imageFiles) > 0 {
		fmt.Println("\n步骤7: 生成视频...")
		finalVideo, err := bg.generateVideo(tempAudioFiles, imageFiles, topic, timestamp)
		if err != nil {
			fmt.Printf("⚠️ 视频生成失败: %v\n", err)
			fmt.Println("继续完成其他步骤...")
		} else {
			blogPost.FinalVideo = finalVideo
			fmt.Printf("✓ 视频生成完成: %s\n", filepath.Base(finalVideo))
		}
	}

	// 8. 清理临时文件（仅在配置允许时）
	fmt.Println("\n步骤8: 清理临时文件...")
	if bg.config.ShouldCleanTempFiles() {
		for _, file := range audioFiles {
			os.Remove(file)
		}
		fmt.Println("✓ 清理完成")
	} else {
		fmt.Println("✓ 保留所有文件（配置要求）")
	}

	// 显示生成结果
	fmt.Println("\n=== 生成完成 ===")
	fmt.Printf("主题: %s\n", blogPost.Topic)
	fmt.Printf("文本文件: %s\n", filepath.Base(textFile))
	fmt.Printf("音频文件: %s\n", filepath.Base(finalAudioPath))
	if blogPost.FinalVideo != "" {
		fmt.Printf("视频文件: %s\n", filepath.Base(blogPost.FinalVideo))
	}
	fmt.Printf("总字数: %d字\n", len(dialogue))
	fmt.Printf("对话段数: %d段\n", len(segments))
	if len(blogPost.GeneratedImages) > 0 {
		fmt.Printf("AI生成图片: %d张\n", len(blogPost.GeneratedImages))
	}
	if bg.sessionDir != "" {
		fmt.Printf("会话目录: %s\n", bg.sessionDir)
	}

	return blogPost, nil
}

// saveAudioFiles 保存音频文件到适当的目录
func (bg *BlogGenerator) saveAudioFiles(audioSegments []tts.AudioSegment, mappingFile *mapping.MappingFile) ([]string, error) {
	var audioFiles []string

	for i, segment := range audioSegments {
		speakerTag := strings.TrimPrefix(segment.Speaker, "[")
		speakerTag = strings.TrimSuffix(speakerTag, "]")

		filename := fmt.Sprintf("segment_%02d_S%s.%s", i+1, speakerTag, bg.config.TTS.Format)

		var filePath string
		if bg.sessionDir != "" {
			filePath = filepath.Join(bg.sessionDir, "audio", filename)
		} else {
			filePath = filepath.Join(bg.config.Output.AudioDir, filename)
		}

		if err := os.WriteFile(filePath, segment.Audio, 0644); err != nil {
			return nil, fmt.Errorf("failed to save audio file %s: %w", filePath, err)
		}

		audioFiles = append(audioFiles, filePath)

		// 添加到映射文件
		preview := segment.Text
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		mappingFile.AddAudio(filePath, segment.Speaker, preview)
	}

	return audioFiles, nil
}

// copyFile 复制文件
func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = destFile.ReadFrom(sourceFile)
	return err
}

func (bg *BlogGenerator) generateVideo(audioFiles []string, imageFiles []string, topic string, timestamp string) (string, error) {
	// 生成输出文件名
	sanitizedTopic := strings.ReplaceAll(strings.ToLower(topic), " ", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "：", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "，", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "。", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "?", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "！", "_")

	outputFilename := fmt.Sprintf("video_%s_%s.mp4",
		sanitizedTopic,
		timestamp)

	var outputPath string
	if bg.sessionDir != "" {
		outputPath = filepath.Join(bg.sessionDir, "video", outputFilename)
	} else {
		outputPath = filepath.Join(bg.config.Output.VideoDir, outputFilename)
	}

	// 生成视频
	if len(audioFiles) == len(imageFiles) {
		// 一对一的情况：每个音频对应一张图片
		err := bg.ffmpeg.GenerateVideoFromSegments(audioFiles, imageFiles, outputPath, &bg.config.Video)
		if err != nil {
			return "", fmt.Errorf("failed to generate video from segments: %w", err)
		}
	} else {
		// 多对多或单张图片的情况
		tempAudioPath := filepath.Join(bg.sessionDir, "temp_audio.mp3")
		err := bg.ffmpeg.MergeAudioFiles(audioFiles, tempAudioPath)
		if err != nil {
			return "", fmt.Errorf("failed to merge audio for video: %w", err)
		}
		defer os.Remove(tempAudioPath)

		err = bg.ffmpeg.GenerateVideoWithAudio(tempAudioPath, imageFiles, outputPath, &bg.config.Video)
		if err != nil {
			return "", fmt.Errorf("failed to generate video with audio: %w", err)
		}
	}

	return outputPath, nil
}

func (bg *BlogGenerator) GenerateMultipleBlogs(topics []string, generateVideo bool, useAIImages bool) ([]*BlogPost, error) {
	blogs := make([]*BlogPost, 0, len(topics))

	for i, topic := range topics {
		fmt.Printf("\n========== 博客 %d/%d ==========\n", i+1, len(topics))

		// 为每个博客创建新的会话
		bg.sessionDir = ""

		blog, err := bg.GenerateSingleBlog(topic, generateVideo, useAIImages)
		if err != nil {
			fmt.Printf("❌ 生成博客失败，主题：'%s'，错误：%v\n", topic, err)
			continue
		}

		blogs = append(blogs, blog)

		// 添加延迟以避免API限制
		if i < len(topics)-1 {
			fmt.Println("\n等待3秒后处理下一个...")
			time.Sleep(3 * time.Second)
		}
	}

	return blogs, nil
}

func (bg *BlogGenerator) GenerateBatch(batchSize int, generateVideo bool, useAIImages bool) error {
	// 优化后的深度主题列表
	topics := []string{
		"人工智能AGI的实现路径及其对人类文明的深远影响",
		"量子计算突破：从实验室到产业化的机遇与挑战",
		"基因编辑技术的伦理边界与未来医学革命",
		"Web3.0与去中心化互联网：重塑数字经济的底层逻辑",
		"碳中和背景下的能源革命：可再生能源与储能技术的创新",
		"脑机接口：人机融合的科技前沿与哲学思考",
		"太空经济：从商业航天到星际殖民的宏伟蓝图",
		"元宇宙：虚拟现实与数字孪生的技术融合与商业模式",
		"合成生物学：设计生命、创造未来的科技革命",
		"区块链的第三次浪潮：DeFi、NFT与DAO的组织创新",
	}

	// 如果batchSize大于主题数量，调整batchSize
	if batchSize > len(topics) {
		batchSize = len(topics)
	}

	selectedTopics := topics[:batchSize]

	fmt.Printf("\n开始批量生成%d个深度博客...\n", batchSize)
	if generateVideo {
		fmt.Printf("视频模式: 启用")
		if useAIImages {
			fmt.Printf(" (AI生成图片)\n")
		} else {
			fmt.Printf(" (使用现有图片)\n")
		}
	} else {
		fmt.Printf("视频模式: 禁用\n")
	}

	startTime := time.Now()
	blogs, err := bg.GenerateMultipleBlogs(selectedTopics, generateVideo, useAIImages)
	if err != nil {
		return fmt.Errorf("批量生成失败: %w", err)
	}
	elapsed := time.Since(startTime)

	// 生成详细的摘要报告
	if err := bg.generateDetailedReport(blogs, elapsed); err != nil {
		return fmt.Errorf("生成报告失败: %w", err)
	}

	fmt.Printf("\n✅ 批量生成完成！\n")
	fmt.Printf("⏱️  总用时: %.1f分钟\n", elapsed.Minutes())
	fmt.Printf("📊 共生成%d个博客\n", len(blogs))
	fmt.Printf("📁 输出目录: %s\n", bg.config.Output.Dir)
	if bg.config.Output.SessionFolders {
		fmt.Printf("📂 会话目录: %s/sessions\n", bg.config.Output.Dir)
	}
	fmt.Printf("📄 报告文件: %s\n", filepath.Join(bg.config.Output.Dir, "batch_report.txt"))

	return nil
}

func (bg *BlogGenerator) generateDetailedReport(blogs []*BlogPost, elapsedTime time.Duration) error {
	reportPath := filepath.Join(bg.config.Output.Dir, "batch_report.txt")

	report := fmt.Sprintf("AI深度博客生成报告\n")
	report += fmt.Sprintf("=" + strings.Repeat("=", 58) + "\n")
	report += fmt.Sprintf("生成时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	report += fmt.Sprintf("用时: %.1f分钟\n", elapsedTime.Minutes())
	report += fmt.Sprintf("生成模型: %s\n", bg.config.SiliconFlow.ChatModel)
	report += fmt.Sprintf("TTS模型: %s\n", bg.config.SiliconFlow.TTSModel)
	if bg.config.Video.Enabled {
		report += fmt.Sprintf("视频生成: 启用\n")
	} else {
		report += fmt.Sprintf("视频生成: 禁用\n")
	}
	report += fmt.Sprintf("保留临时文件: %t\n", bg.config.Output.KeepTempFiles)
	report += fmt.Sprintf("使用会话文件夹: %t\n", bg.config.Output.SessionFolders)
	report += fmt.Sprintf("博客总数: %d\n\n", len(blogs))

	report += "生成详情:\n"
	report += strings.Repeat("-", 60) + "\n"

	totalSegments := 0
	totalWords := 0
	videoCount := 0
	aiImageCount := 0

	for i, blog := range blogs {
		segments := len(blog.AudioSegments)
		words := len(blog.Dialogue)
		totalSegments += segments
		totalWords += words

		// 估算音频时长（假设每分钟150字）
		estimatedMinutes := words / 150
		if estimatedMinutes < 1 {
			estimatedMinutes = 1
		}

		if blog.FinalVideo != "" {
			videoCount++
		}
		if len(blog.GeneratedImages) > 0 {
			aiImageCount += len(blog.GeneratedImages)
		}

		report += fmt.Sprintf("\n[%d] %s\n", i+1, blog.Topic)
		if blog.SessionDir != "" {
			report += fmt.Sprintf("    会话目录: %s\n", blog.SessionDir)
		}
		report += fmt.Sprintf("    文本: %s\n", filepath.Base(blog.TextFile))
		report += fmt.Sprintf("    音频: %s\n", filepath.Base(blog.FinalAudio))
		if blog.FinalVideo != "" {
			report += fmt.Sprintf("    视频: %s\n", filepath.Base(blog.FinalVideo))
		}
		if len(blog.GeneratedImages) > 0 {
			report += fmt.Sprintf("    AI图片: %d张\n", len(blog.GeneratedImages))
		}
		report += fmt.Sprintf("    字数: %d字 | 时长: ~%d分钟\n", words, estimatedMinutes)
		report += fmt.Sprintf("    段落: %d段 | 语音: %s/%s\n",
			segments,
			filepath.Base(bg.config.TTS.Voice1),
			filepath.Base(bg.config.TTS.Voice2))
	}

	report += fmt.Sprintf("\n" + strings.Repeat("=", 60) + "\n")
	report += fmt.Sprintf("统计摘要:\n")
	report += fmt.Sprintf("- 总对话段落数: %d\n", totalSegments)
	report += fmt.Sprintf("- 总字数: %d字\n", totalWords)
	report += fmt.Sprintf("- 平均每博客: %.1f分钟\n", float64(totalWords)/float64(len(blogs))/150)
	report += fmt.Sprintf("- 平均段落数: %.1f段\n", float64(totalSegments)/float64(len(blogs)))
	report += fmt.Sprintf("- 视频生成数: %d/%d\n", videoCount, len(blogs))
	report += fmt.Sprintf("- AI生成图片总数: %d\n", aiImageCount)

	return os.WriteFile(reportPath, []byte(report), 0644)
}