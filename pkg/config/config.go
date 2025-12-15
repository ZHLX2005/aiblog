package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	SiliconFlow SiliconFlowConfig `mapstructure:"siliconflow"`
	TTS         TTSConfig         `mapstructure:"tts"`
	FFmpeg      FFmpegConfig      `mapstructure:"ffmpeg"`
	Video       VideoConfig       `mapstructure:"video"`
	Output      OutputConfig      `mapstructure:"output"`
}

type SiliconFlowConfig struct {
	APIKey      string `mapstructure:"api_key"`
	BaseURL     string `mapstructure:"base_url"`
	ChatModel   string `mapstructure:"chat_model"`
	TTSModel    string `mapstructure:"tts_model"`
}

type TTSConfig struct {
	Voice1      string  `mapstructure:"voice1"`
	Voice2      string  `mapstructure:"voice2"`
	SampleRate  int     `mapstructure:"sample_rate"`
	Speed       float64 `mapstructure:"speed"`
	Gain        float64 `mapstructure:"gain"`
	Format      string  `mapstructure:"format"`
	MaxTokens   int     `mapstructure:"max_tokens"`
}

type FFmpegConfig struct {
	Path         string `mapstructure:"path"`
	OutputCodec  string `mapstructure:"output_codec"`
	OutputFormat string `mapstructure:"output_format"`
	VideoCodec   string `mapstructure:"video_codec"`
}

type VideoConfig struct {
	Enabled       bool     `mapstructure:"enabled"`
	Width         int      `mapstructure:"width"`
	Height        int      `mapstructure:"height"`
	FPS           int      `mapstructure:"fps"`
	ImageDir      string   `mapstructure:"image_dir"`
	AutoImages    bool     `mapstructure:"auto_images"`
	ImagePatterns []string `mapstructure:"image_patterns"`
	SlideDuration float64  `mapstructure:"slide_duration"`
	Quality       string   `mapstructure:"quality"`
	PixelFormat   string   `mapstructure:"pixel_format"`
}

type OutputConfig struct {
	Dir            string `mapstructure:"dir"`
	AudioDir       string `mapstructure:"audio_dir"`
	VideoDir       string `mapstructure:"video_dir"`
	KeepTempFiles  bool   `mapstructure:"keep_temp_files"`
	SessionFolders bool   `mapstructure:"session_folders"`
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// 设置默认值
	setDefaults()

	// 从环境变量读取配置
	viper.AutomaticEnv()
	viper.SetEnvPrefix("AIBLOG")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			fmt.Println("Config file not found, using defaults and environment variables")
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("unable to decode config: %w", err)
	}

	// 验证必要的配置
	if err := validate(&config); err != nil {
		return nil, err
	}

	// 创建输出目录
	if err := createDirectories(&config); err != nil {
		return nil, fmt.Errorf("failed to create directories: %w", err)
	}

	return &config, nil
}

// CreateSessionDir 为每次请求创建独立的会话目录
func (c *Config) CreateSessionDir() (string, error) {
	if !c.Output.SessionFolders {
		return "", nil
	}

	sessionID := time.Now().Format("20060102_150405")
	sessionDir := filepath.Join(c.Output.Dir, "sessions", sessionID)

	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create session directory: %w", err)
	}

	return sessionDir, nil
}

// GetAudioPath 获取音频文件的存储路径
func (c *Config) GetAudioPath(sessionDir string, filename string) string {
	if c.Output.SessionFolders && sessionDir != "" {
		return filepath.Join(sessionDir, "audio", filename)
	}
	return filepath.Join(c.Output.AudioDir, filename)
}

// GetVideoPath 获取视频文件的存储路径
func (c *Config) GetVideoPath(sessionDir string, filename string) string {
	if c.Output.SessionFolders && sessionDir != "" {
		return filepath.Join(sessionDir, "video", filename)
	}
	return filepath.Join(c.Output.VideoDir, filename)
}

// GetImagePath 获取图片文件的存储路径
func (c *Config) GetImagePath(sessionDir string, filename string) string {
	if c.Output.SessionFolders && sessionDir != "" {
		return filepath.Join(sessionDir, "images", filename)
	}
	return filepath.Join(c.Output.VideoDir, filename)
}

// ShouldCleanTempFiles 判断是否应该清理临时文件
func (c *Config) ShouldCleanTempFiles() bool {
	return !c.Output.KeepTempFiles
}

func setDefaults() {
	// SiliconFlow默认配置
	viper.SetDefault("siliconflow.base_url", "https://api.siliconflow.cn/v1")
	viper.SetDefault("siliconflow.chat_model", "deepseek-ai/DeepSeek-V3.2-Exp")
	viper.SetDefault("siliconflow.tts_model", "fnlp/MOSS-TTSD-v0.5")

	// TTS默认配置
	viper.SetDefault("tts.voice1", "fnlp/MOSS-TTSD-v0.5:alex")
	viper.SetDefault("tts.voice2", "fnlp/MOSS-TTSD-v0.5:anna")
	viper.SetDefault("tts.sample_rate", 32000)
	viper.SetDefault("tts.speed", 1.0)
	viper.SetDefault("tts.gain", 0)
	viper.SetDefault("tts.format", "mp3")
	viper.SetDefault("tts.max_tokens", 2048)

	// FFmpeg默认配置
	viper.SetDefault("ffmpeg.path", "ffmpeg")
	viper.SetDefault("ffmpeg.output_codec", "mp3")
	viper.SetDefault("ffmpeg.output_format", "mp3")
	viper.SetDefault("ffmpeg.video_codec", "libx264")

	// Video默认配置
	viper.SetDefault("video.enabled", false)
	viper.SetDefault("video.width", 1920)
	viper.SetDefault("video.height", 1080)
	viper.SetDefault("video.fps", 30)
	viper.SetDefault("video.image_dir", "./test")
	viper.SetDefault("video.auto_images", true)
	viper.SetDefault("video.image_patterns", []string{"*.png", "*.jpg", "*.jpeg"})
	viper.SetDefault("video.slide_duration", 5.0)
	viper.SetDefault("video.quality", "high")
	viper.SetDefault("video.pixel_format", "yuv420p")

	// 输出目录配置
	viper.SetDefault("output.dir", "./output")
	viper.SetDefault("output.audio_dir", "./output/audio")
	viper.SetDefault("output.video_dir", "./output/video")
	viper.SetDefault("output.keep_temp_files", true) // 默认保留临时文件
	viper.SetDefault("output.session_folders", true)  // 默认使用会话文件夹
}

func validate(config *Config) error {
	if config.SiliconFlow.APIKey == "" {
		config.SiliconFlow.APIKey = os.Getenv("SILICONFLOW_API_KEY")
	}
	if config.SiliconFlow.APIKey == "" {
		return fmt.Errorf("SiliconFlow API key is required")
	}

	if config.Video.Enabled {
		if config.Video.Width <= 0 || config.Video.Height <= 0 {
			return fmt.Errorf("video dimensions must be positive")
		}
		if config.Video.FPS <= 0 {
			return fmt.Errorf("video FPS must be positive")
		}
	}

	return nil
}

func createDirectories(config *Config) error {
	dirs := []string{
		config.Output.Dir,
		config.Output.AudioDir,
	}

	if config.Video.Enabled {
		dirs = append(dirs, config.Output.VideoDir)
	}

	// 如果启用会话文件夹，创建sessions目录
	if config.Output.SessionFolders {
		dirs = append(dirs, filepath.Join(config.Output.Dir, "sessions"))
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}