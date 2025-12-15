package images

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiblog/pkg/config"
	"aiblog/pkg/imagegen"
	"aiblog/pkg/tts"
)

type ImageManager struct {
	config      *config.VideoConfig
	imageClient *imagegen.ImageGenClient
}

func NewImageManager(videoConfig *config.VideoConfig, sfConfig *config.SiliconFlowConfig) *ImageManager {
	return &ImageManager{
		config:      videoConfig,
		imageClient: imagegen.NewImageGenClient(sfConfig),
	}
}

func (im *ImageManager) GetAvailableImages() ([]string, error) {
	var images []string

	// 确保图片目录存在
	if _, err := os.Stat(im.config.ImageDir); os.IsNotExist(err) {
		return images, fmt.Errorf("image directory not found: %s", im.config.ImageDir)
	}

	// 遍历图片目录
	err := filepath.Walk(im.config.ImageDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录
		if info.IsDir() {
			return nil
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(path))
		for _, pattern := range im.config.ImagePatterns {
			if strings.Contains(pattern, ext) {
				images = append(images, path)
				break
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan image directory: %w", err)
	}

	if len(images) == 0 {
		return nil, fmt.Errorf("no images found in directory %s", im.config.ImageDir)
	}

	return images, nil
}

func (im *ImageManager) AssignImagesToSegments(segmentCount int) ([]string, error) {
	images, err := im.GetAvailableImages()
	if err != nil {
		return nil, err
	}

	// 如果没有启用自动图片分配，返回空
	if !im.config.AutoImages {
		return nil, nil
	}

	// 如果图片数量少于段落数，循环使用图片
	assignedImages := make([]string, segmentCount)
	for i := 0; i < segmentCount; i++ {
		imageIndex := i % len(images)
		assignedImages[i] = images[imageIndex]
	}

	return assignedImages, nil
}

func (im *ImageManager) ValidateImages(imagePaths []string) error {
	for _, imagePath := range imagePaths {
		if _, err := os.Stat(imagePath); os.IsNotExist(err) {
			return fmt.Errorf("image file not found: %s", imagePath)
		}

		// 检查文件扩展名
		ext := strings.ToLower(filepath.Ext(imagePath))
		valid := false
		for _, pattern := range im.config.ImagePatterns {
			if strings.Contains(pattern, ext) {
				valid = true
				break
			}
		}

		if !valid {
			return fmt.Errorf("invalid image format: %s", imagePath)
		}
	}

	return nil
}

func (im *ImageManager) CreateDefaultImages(segmentCount int) ([]string, error) {
	// 这里可以集成AI图片生成API或创建默认图片
	// 目前先返回错误，提示用户添加图片
	return nil, fmt.Errorf("no images available. Please add images to %s or enable auto_images", im.config.ImageDir)
}

// GenerateImagesForSegments 使用AI为每个段落生成对应的PPT风格图片
func (im *ImageManager) GenerateImagesForSegments(segments []tts.AudioSegment, topic string, outputDir string) ([]string, error) {
	fmt.Printf("\n=== AI图片生成 ===\n")
	fmt.Printf("主题: %s\n", topic)
	fmt.Printf("段落数: %d\n", len(segments))
	fmt.Println("-------------------")

	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	// 创建生成图片的专用目录
	timestamp := time.Now().Format("20060102_150405")
	genImageDir := filepath.Join(outputDir, fmt.Sprintf("generated_images_%s", timestamp))
	if err := os.MkdirAll(genImageDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create generated images directory: %w", err)
	}

	// 准备文本段落
	textSegments := make([]imagegen.TextSegment, len(segments))
	for i, segment := range segments {
		textSegments[i] = imagegen.TextSegment{
			Index:   i,
			Text:    segment.Text,
			Speaker: segment.Speaker,
		}
	}

	// 生成图片
	imagePaths, err := im.imageClient.GenerateImagesForSegments(textSegments, topic, genImageDir)
	if err != nil {
		return nil, fmt.Errorf("failed to generate images: %w", err)
	}

	fmt.Printf("\n✅ 图片生成完成，共%d张\n", len(imagePaths))
	fmt.Printf("📁 保存位置: %s\n", genImageDir)

	return imagePaths, nil
}

// GetImagesForSegments 获取图片，优先使用AI生成的，如果未启用则使用现有图片
func (im *ImageManager) GetImagesForSegments(segments []tts.AudioSegment, topic string,
	useAIImages bool, outputDir string) ([]string, error) {

	// 如果启用AI图片生成
	if useAIImages {
		return im.GenerateImagesForSegments(segments, topic, outputDir)
	}

	// 否则使用现有图片
	images, err := im.AssignImagesToSegments(len(segments))
	if err != nil {
		return nil, fmt.Errorf("failed to assign existing images: %w", err)
	}

	return images, nil
}

// CleanupGeneratedImages 清理生成的临时图片文件
func (im *ImageManager) CleanupGeneratedImages(imagePaths []string) error {
	for _, imagePath := range imagePaths {
		// 只清理generated_images_*目录下的文件
		dir := filepath.Dir(imagePath)
		if strings.Contains(filepath.Base(dir), "generated_images_") {
			if err := os.Remove(imagePath); err != nil {
				fmt.Printf("警告: 无法删除图片文件 %s: %v\n", imagePath, err)
			}
		}
	}

	return nil
}