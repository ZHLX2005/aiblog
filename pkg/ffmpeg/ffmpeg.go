package ffmpeg

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"aiblog/pkg/config"
)

type FFmpegProcessor struct {
	config *config.FFmpegConfig
}

type VideoConfig struct {
	config.VideoConfig
}

func NewFFmpegProcessor(config *config.FFmpegConfig) *FFmpegProcessor {
	return &FFmpegProcessor{
		config: config,
	}
}

func (f *FFmpegProcessor) CheckFFmpeg() error {
	cmd := exec.Command(f.config.Path, "-version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg not found or not accessible: %w", err)
	}
	return nil
}

func (f *FFmpegProcessor) CreateFileList(audioFiles []string) (string, error) {
	// 创建临时文件列表
	tmpFile, err := ioutil.TempFile("", "ffmpeg_list_*.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer tmpFile.Close()

	// 写入音频文件路径
	for _, audioFile := range audioFiles {
		// 确保路径是绝对路径
		absPath, err := filepath.Abs(audioFile)
		if err != nil {
			return "", fmt.Errorf("failed to get absolute path for %s: %w", audioFile, err)
		}

		// Windows路径需要替换反斜杠
		absPath = strings.ReplaceAll(absPath, "\\", "/")

		// 引号包裹路径，处理空格问题
		if _, err := fmt.Fprintf(tmpFile, "file '%s'\n", absPath); err != nil {
			return "", fmt.Errorf("failed to write to temp file: %w", err)
		}
	}

	return tmpFile.Name(), nil
}

func (f *FFmpegProcessor) MergeAudioFiles(audioFiles []string, outputPath string) error {
	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files to merge")
	}

	if len(audioFiles) == 1 {
		// 如果只有一个文件，直接复制
		return f.copyFile(audioFiles[0], outputPath)
	}

	// 创建文件列表
	listFile, err := f.CreateFileList(audioFiles)
	if err != nil {
		return err
	}
	defer os.Remove(listFile)

	// 构建ffmpeg命令
	cmd := exec.Command(f.config.Path,
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y", // 覆盖输出文件
		outputPath,
	)

	// 捕获输出
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// 执行命令
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg failed: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

func (f *FFmpegProcessor) GenerateVideoSegment(imagePath, audioPath, outputPath string, videoConfig *config.VideoConfig) error {
	// 检查文件是否存在
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("image file not found: %s", imagePath)
	}
	if _, err := os.Stat(audioPath); os.IsNotExist(err) {
		return fmt.Errorf("audio file not found: %s", audioPath)
	}

	// 构建ffmpeg命令
	args := []string{
		"-loop", "1", // 循环图片
		"-i", imagePath,
		"-i", audioPath,
		"-c:v", f.config.VideoCodec, // 视频编码器
		"-c:a", f.config.OutputCodec, // 音频编码器
		"-pix_fmt", videoConfig.PixelFormat,
		"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d",
			videoConfig.Width, videoConfig.Height, videoConfig.Width, videoConfig.Height),
		"-r", fmt.Sprintf("%d", videoConfig.FPS), // 帧率
		"-shortest", // 视频长度以最短的轨道为准（音频）
		"-y", // 覆盖输出文件
		outputPath,
	}

	// 根据质量设置调整参数
	switch videoConfig.Quality {
	case "high":
		args = append(args[:12], append([]string{"-crf", "18"}, args[12:]...)...)
	case "medium":
		args = append(args[:12], append([]string{"-crf", "23"}, args[12:]...)...)
	case "low":
		args = append(args[:12], append([]string{"-crf", "28"}, args[12:]...)...)
	}

	cmd := exec.Command(f.config.Path, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create video segment: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

func (f *FFmpegProcessor) MergeVideoSegments(videoFiles []string, outputPath string) error {
	if len(videoFiles) == 0 {
		return fmt.Errorf("no video files to merge")
	}

	if len(videoFiles) == 1 {
		return f.copyFile(videoFiles[0], outputPath)
	}

	// 创建文件列表
	listFile, err := f.CreateFileList(videoFiles)
	if err != nil {
		return err
	}
	defer os.Remove(listFile)

	// 合并视频文件
	cmd := exec.Command(f.config.Path,
		"-f", "concat",
		"-safe", "0",
		"-i", listFile,
		"-c", "copy",
		"-y",
		outputPath,
	)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to merge video segments: %w\nstderr: %s", err, stderr.String())
	}

	return nil
}

func (f *FFmpegProcessor) GenerateVideoFromSegments(audioFiles []string, imageFiles []string, outputPath string, videoConfig *config.VideoConfig) error {
	if len(audioFiles) == 0 {
		return fmt.Errorf("no audio files provided")
	}

	if len(imageFiles) == 0 {
		return fmt.Errorf("no image files provided")
	}

	// 确保图片数量足够，如果不够则循环使用
	for len(imageFiles) < len(audioFiles) {
		imageFiles = append(imageFiles, imageFiles...)
	}

	// 创建临时目录存储视频片段
	tempDir, err := ioutil.TempDir("", "video_segments_")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 为每个音频文件生成对应的视频片段
	videoSegments := make([]string, len(audioFiles))
	for i := 0; i < len(audioFiles); i++ {
		audioFile := audioFiles[i]
		imageFile := imageFiles[i]

		segmentPath := filepath.Join(tempDir, fmt.Sprintf("segment_%03d.mp4", i))
		videoSegments[i] = segmentPath

		if err := f.GenerateVideoSegment(imageFile, audioFile, segmentPath, videoConfig); err != nil {
			return fmt.Errorf("failed to generate video segment %d: %w", i, err)
		}
	}

	// 合并所有视频片段
	if err := f.MergeVideoSegments(videoSegments, outputPath); err != nil {
		return fmt.Errorf("failed to merge video segments: %w", err)
	}

	return nil
}

func (f *FFmpegProcessor) GenerateVideoWithAudio(audioPath string, imageFiles []string, outputPath string, videoConfig *config.VideoConfig) error {
	// 先将音频文件合并为一个（如果有多个）
	if len(audioPath) == 0 {
		return fmt.Errorf("no audio path provided")
	}

	// 创建临时目录
	tempDir, err := ioutil.TempDir("", "video_generation_")
	if err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// 如果有多个图片，创建图片序列
	if len(imageFiles) > 1 {
		// 创建concat文件
		concatFile := filepath.Join(tempDir, "images.txt")
		file, err := os.Create(concatFile)
		if err != nil {
			return fmt.Errorf("failed to create concat file: %w", err)
		}
		defer file.Close()

		// 计算每张图片的显示时长
		for _, imagePath := range imageFiles {
			if _, err := fmt.Fprintf(file, "file '%s'\nduration %f\n",
				strings.ReplaceAll(imagePath, "\\", "/"), videoConfig.SlideDuration); err != nil {
				return fmt.Errorf("failed to write to concat file: %w", err)
			}
		}

		// 使用concat demuxer创建视频流
		cmd := exec.Command(f.config.Path,
			"-f", "concat",
			"-safe", "0",
			"-i", concatFile,
			"-i", audioPath,
			"-c:v", f.config.VideoCodec,
			"-c:a", f.config.OutputCodec,
			"-pix_fmt", videoConfig.PixelFormat,
			"-vf", fmt.Sprintf("scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d",
				videoConfig.Width, videoConfig.Height, videoConfig.Width, videoConfig.Height),
			"-r", fmt.Sprintf("%d", videoConfig.FPS),
			"-shortest",
			"-y",
			outputPath,
		)

		var stderr bytes.Buffer
		cmd.Stderr = &stderr

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to generate video with image sequence: %w\nstderr: %s", err, stderr.String())
		}
	} else {
		// 单张图片的情况
		return f.GenerateVideoSegment(imageFiles[0], audioPath, outputPath, videoConfig)
	}

	return nil
}

func (f *FFmpegProcessor) GeneratePodcast(audioFiles []string, topic string, outputDir string) (string, error) {
	if err := f.CheckFFmpeg(); err != nil {
		return "", err
	}

	// 生成输出文件名
	timestamp := time.Now().Format("20060102_150405")
	sanitizedTopic := strings.ReplaceAll(strings.ToLower(topic), " ", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "：", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "，", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "。", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "?", "_")
	sanitizedTopic = strings.ReplaceAll(sanitizedTopic, "！", "_")

	outputFilename := fmt.Sprintf("podcast_%s_%s.%s",
		sanitizedTopic,
		timestamp,
		f.config.OutputFormat)

	outputPath := filepath.Join(outputDir, outputFilename)

	// 合并音频文件
	if err := f.MergeAudioFiles(audioFiles, outputPath); err != nil {
		return "", fmt.Errorf("failed to merge audio files: %w", err)
	}

	return outputPath, nil
}

func (f *FFmpegProcessor) copyFile(src, dst string) error {
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
	if err != nil {
		return err
	}

	return nil
}