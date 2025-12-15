package cmd

import (
	"fmt"
	"os"

	"aiblog/pkg/blog"
	"aiblog/pkg/config"

	"github.com/spf13/cobra"
)

var topic string
var video bool
var aiImages bool

// generateCmd represents the generate command
var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "生成单个AI博客",
	Long: `根据指定的主题生成一个AI博客，包括：
1. 生成主持人和嘉宾的对话内容
2. 将对话转换为不同语音的音频
3. 合成完整的播客音频文件
4. 可选：生成对应的视频文件（图片+音频）
5. 可选：使用AI自动生成PPT风格图片`,
	Run: func(cmd *cobra.Command, args []string) {
		if topic == "" {
			fmt.Println("错误: 请提供博客主题 (-t/--topic)")
			os.Exit(1)
		}

		// 加载配置
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			os.Exit(1)
		}

		// 检查视频生成要求
		if video && !cfg.Video.Enabled {
			fmt.Println("警告: 视频生成功能在配置中已禁用，将只生成音频")
			video = false
		}

		// 如果要求AI生成图片但视频功能未启用
		if aiImages && !video {
			fmt.Println("警告: AI图片生成需要启用视频模式 (-v/--video)")
			aiImages = false
		}

		// 创建博客生成器
		generator := blog.NewBlogGenerator(cfg)

		// 生成博客
		blogPost, err := generator.GenerateSingleBlog(topic, video, aiImages)
		if err != nil {
			fmt.Printf("生成博客失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n=== 博客生成完成 ===")
		fmt.Printf("主题: %s\n", blogPost.Topic)
		fmt.Printf("对话文本: %s\n", blogPost.TextFile)
		fmt.Printf("音频文件: %s\n", blogPost.FinalAudio)
		if blogPost.FinalVideo != "" {
			fmt.Printf("视频文件: %s\n", blogPost.FinalVideo)
		}
		if len(blogPost.GeneratedImages) > 0 {
			fmt.Printf("AI生成图片: %d张\n", len(blogPost.GeneratedImages))
		}
		fmt.Printf("对话段落数: %d\n", len(blogPost.AudioSegments))
		if blogPost.SessionDir != "" {
			fmt.Printf("会话目录: %s\n", blogPost.SessionDir)
		}
	},
}

func init() {
	rootCmd.AddCommand(generateCmd)

	generateCmd.Flags().StringVarP(&topic, "topic", "t", "", "博客主题")
	generateCmd.Flags().BoolVarP(&video, "video", "v", false, "同时生成视频文件")
	generateCmd.Flags().BoolVarP(&aiImages, "ai-images", "a", false, "使用AI生成PPT风格图片（需要启用视频模式）")
	generateCmd.MarkFlagRequired("topic")
}