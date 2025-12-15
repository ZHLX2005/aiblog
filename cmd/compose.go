package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"aiblog/pkg/composer"
	"aiblog/pkg/config"

	"github.com/spf13/cobra"
)

var (
	audioDir     string
	imageDir     string
	outputDir    string
	pattern      string
	batchSize    int
	tomlFile     string
)

// composeCmd represents the compose command
var composeCmd = &cobra.Command{
	Use:   "compose",
	Short: "合成视频 - 自定义图片和音频绑定",
	Long: `使用自定义规则将音频和图片合成为视频。
支持多种模式：
1. TOML模式：使用映射文件精确控制音频和图片对应关系（推荐）
2. 目录模式：自动匹配目录中的音频和图片文件
3. 自定义模式：使用指定的模式匹配图片和音频
4. 批量模式：分批处理大量文件

示例：
  # 使用TOML映射文件（推荐）
  blog compose -t mapping.toml

  # 使用默认规则（p1对应S1，p2对应S2）
  blog compose -a ./audio -i ./images -o ./output

  # 使用自定义模式
  blog compose -a ./audio -i ./images -p "bg1,bg2" -o ./output

  # 批量处理
  blog compose -a ./audio -i ./images -b 5 -p "p1,p2" -o ./output`,
	Run: func(cmd *cobra.Command, args []string) {
		// 加载配置
		cfg, err := config.Load()
		if err != nil {
			fmt.Printf("加载配置失败: %v\n", err)
			os.Exit(1)
		}

		// 创建视频合成器
		vc := composer.NewVideoComposer(&cfg.Video, &cfg.FFmpeg)

		// 检查是否使用TOML模式
		if tomlFile != "" {
			// TOML模式
			if _, err := os.Stat(tomlFile); os.IsNotExist(err) {
				fmt.Printf("错误: TOML映射文件不存在: %s\n", tomlFile)
				os.Exit(1)
			}

			if err := vc.ComposeFromTOML(tomlFile); err != nil {
				fmt.Printf("TOML模式合成失败: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// 传统模式验证
		if audioDir == "" || imageDir == "" {
			fmt.Println("错误: 必须指定TOML映射文件 (-t) 或同时指定音频目录 (-a) 和图片目录 (-i)")
			os.Exit(1)
		}

		// 检查目录是否存在
		if _, err := os.Stat(audioDir); os.IsNotExist(err) {
			fmt.Printf("错误: 音频目录不存在: %s\n", audioDir)
			os.Exit(1)
		}
		if _, err := os.Stat(imageDir); os.IsNotExist(err) {
			fmt.Printf("错误: 图片目录不存在: %s\n", imageDir)
			os.Exit(1)
		}

		// 设置默认输出目录
		if outputDir == "" {
			outputDir = "./output"
		}

		// 确保输出目录存在
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			fmt.Printf("创建输出目录失败: %v\n", err)
			os.Exit(1)
		}

		// 根据参数选择合成模式
		if batchSize > 0 {
			// 批量模式
			fmt.Printf("\n批量合成模式启动...\n")
			if pattern == "" {
				pattern = "p1,p2" // 默认模式
			}

			if err := vc.BatchCompose(audioDir, imageDir, outputDir, batchSize, pattern); err != nil {
				fmt.Printf("批量合成失败: %v\n", err)
				os.Exit(1)
			}
		} else {
			// 单个视频合成
			outputPath := filepath.Join(outputDir, "composed_video.mp4")

			if pattern != "" {
				// 自定义模式
				fmt.Printf("\n使用自定义模式: %s\n", pattern)
				if err := vc.ComposeWithCustomPattern(audioDir, imageDir, outputPath, pattern); err != nil {
					fmt.Printf("自定义模式合成失败: %v\n", err)
					os.Exit(1)
				}
			} else {
				// 默认目录模式
				fmt.Printf("\n使用默认规则: p1对应S1，p2对应S2\n")
				if err := vc.ComposeFromDirectories(audioDir, imageDir, outputPath, nil); err != nil {
					fmt.Printf("目录模式合成失败: %v\n", err)
					os.Exit(1)
				}
			}

			fmt.Printf("\n视频合成完成: %s\n", outputPath)
		}
	},
}

func init() {
	rootCmd.AddCommand(composeCmd)

	composeCmd.Flags().StringVarP(&tomlFile, "toml", "t", "", "TOML映射文件路径（推荐使用）")
	composeCmd.Flags().StringVarP(&audioDir, "audio", "a", "", "音频文件目录")
	composeCmd.Flags().StringVarP(&imageDir, "images", "i", "", "图片文件目录")
	composeCmd.Flags().StringVarP(&outputDir, "output", "o", "", "输出目录")
	composeCmd.Flags().StringVarP(&pattern, "pattern", "p", "", "图片匹配模式（如: p1,p2）")
	composeCmd.Flags().IntVarP(&batchSize, "batch", "b", 0, "批量处理大小（0表示不使用批量）")
}