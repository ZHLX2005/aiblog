package cmd

import (
	"fmt"
	"os"

	"aiblog/pkg/blog"
	"aiblog/pkg/config"

	"github.com/spf13/cobra"
)

var count int
var batchVideo bool
var batchAIImages bool

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "批量生成多个AI博客",
	Long: `批量生成多个不同主题的AI博客。程序会使用预定义的主题列表，
生成指定数量的博客，每个博客包括对话文本、音频文件，以及可选的视频文件。
支持使用AI自动生成PPT风格图片。`,
	Run: func(cmd *cobra.Command, args []string) {
		// 加载配置
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			os.Exit(1)
		}

		// 验证batch size
		if count <= 0 || count > 10 {
			fmt.Println("错误: 批量数量必须在1-10之间")
			os.Exit(1)
		}

		// 检查视频生成要求
		if batchVideo && !cfg.Video.Enabled {
			fmt.Println("警告: 视频生成功能在配置中已禁用，将只生成音频")
			batchVideo = false
		}

		// 如果要求AI生成图片但视频功能未启用
		if batchAIImages && !batchVideo {
			fmt.Println("警告: AI图片生成需要启用视频模式 (-v/--video)")
			batchAIImages = false
		}

		// 创建博客生成器
		generator := blog.NewBlogGenerator(cfg)

		// 批量生成博客
		if err := generator.GenerateBatch(count, batchVideo, batchAIImages); err != nil {
			fmt.Printf("批量生成博客失败: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n所有博客已生成完成！")
		fmt.Printf("查看输出目录: %s\n", cfg.Output.Dir)
		if cfg.Output.SessionFolders {
			fmt.Printf("会话目录: %s/sessions\n", cfg.Output.Dir)
		}
		if batchVideo {
			fmt.Printf("视频目录: %s\n", cfg.Output.VideoDir)
		}
	},
}

func init() {
	rootCmd.AddCommand(batchCmd)

	batchCmd.Flags().IntVarP(&count, "count", "c", 5, "生成的博客数量 (1-10)")
	batchCmd.Flags().BoolVarP(&batchVideo, "video", "v", false, "同时生成视频文件")
	batchCmd.Flags().BoolVarP(&batchAIImages, "ai-images", "a", false, "使用AI生成PPT风格图片（需要启用视频模式）")
}