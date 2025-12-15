# AI博客生成器 🤖

[![Go Version](https://img.shields.io/badge/Go-1.19+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![FFmpeg](https://img.shields.io/badge/FFmpeg-4.0+-brightgreen.svg)](https://ffmpeg.org)

一个基于AI的自动化博客生成系统，能够：
- 生成深度对话内容
- 转换为多角色语音播客
- 创建带有AI生成图片的视频内容

> ✨ **特色功能**: 智能TOML映射文件系统，精确控制音视频对应关系

## 🌟 核心特性

### 📝 内容生成
- **深度对话**: 基于DeepSeek-V3模型生成主持人与嘉宾的专业对话
- **智能分段**: 自动识别对话结构，分离不同说话者的内容
- **主题丰富**: 支持科技、教育、文化等多领域主题

### 🎙️ 音频处理
- **多角色TTS**: 支持MOSS-TTS等文本转语音引擎
- **语音差异化**: 为不同角色分配不同音色
- **高质量输出**: 支持MP3、WAV等主流音频格式

### 🎨 视频制作
- **AI图片生成**: 集成SiliconFlow图像生成API，创建PPT风格插图
- **智能匹配**: 自动将对话内容与相关图片配对
- **灵活组合**: 支持一对多、多对一的音频图片组合

### 📁 文件管理
- **会话隔离**: 每次生成创建独立会话文件夹
- **TOML映射**: 生成TOML配置文件，精确控制音视频映射
- **保留选项**: 可选择保留或清理临时文件

### 🛠️ 灵活部署
- **多种模式**: 支持单篇、批量、自定义组合生成
- **命令行界面**: 简洁易用的CLI工具
- **配置驱动**: 通过YAML文件灵活配置

## 🚀 快速开始

### 环境要求

- Go 1.19+
- FFmpeg 4.0+
- SiliconFlow API密钥

### 安装步骤

1. **克隆项目**
```bash
git clone https://github.com/yourusername/aiblog.git
cd aiblog
```

2. **安装依赖**
```bash
go mod tidy
```

3. **配置文件**
```bash
cp config.example.yaml config.yaml
# 编辑 config.yaml，填入您的API密钥
```

4. **构建项目**
```bash
go build -o blog.exe
```

### 基础使用

1. **生成单篇博客**
```bash
# 仅生成音频
./blog.exe generate -t "人工智能的未来发展"

# 生成音频 + 视频（使用现有图片）
./blog.exe generate -t "量子计算基础" -v

# 生成音频 + 视频（使用AI生成图片）
./blog.exe generate -t "区块链技术原理" -v -a
```

2. **批量生成**
```bash
# 生成5篇深度博客
./blog.exe batch 5 -v -a
```

3. **自定义视频合成**
```bash
# 使用TOML映射文件（推荐）
./blog.exe compose -t session1/ai_tech_mapping.toml

# 使用目录和模式
./blog.exe compose -a ./audio -i ./images -p "p1,p2" -o ./output

# 批量处理
./blog.exe compose -a ./audio -i ./images -b 5 -p "bg1,bg2"
```

## 📖 详细配置

### config.yaml 配置说明

```yaml
# SiliconFlow API配置
siliconflow:
  api_key: "your-api-key-here"     # 必需：您的API密钥
  base_url: "https://api.siliconflow.cn"  # API基础地址
  chat_model: "deepseek-chat"      # 对话模型
  tts_model: "speech-gh-chinese"   # TTS模型
  image_model: "stabilityai/stable-diffusion-3-5-large"  # 图像生成模型

# TTS配置
tts:
  format: "mp3"                    # 输出格式
  voice1: "male"                   # S1音色
  voice2: "female"                 # S2音色
  speed: 1.0                       # 语速

# 视频配置
video:
  enabled: true                    # 启用视频生成
  image_patterns: ["*.jpg", "*.png", "*.jpeg"]  # 支持的图片格式
  fps: 24                          # 视频帧率
  resolution: "1920x1080"          # 视频分辨率

# FFmpeg配置
ffmpeg:
  binary_path: "ffmpeg"            # FFmpeg可执行文件路径
  output_format: "mp3"             # 音频输出格式

# 输出配置
output:
  dir: "./output"                  # 主输出目录
  video_dir: "./output/videos"     # 视频输出目录
  audio_dir: "./output/audio"      # 音频输出目录
  keep_temp_files: true            # 保留临时文件
  session_folders: true            # 使用会话文件夹
```

### TOML映射文件格式

生成博客时会自动创建TOML映射文件，示例格式：

```toml
title = "人工智能的发展前景"
description = "AI生成的博客对话，请为每个音频文件指定对应的图片"
output_path = "./composed_video.mp4"
audio_count = 6

[[audio_files]]
  audio_file = "/path/to/segment_01_SS1.mp3"
  image_file = "/path/to/image1.jpg"  # 用户需要填写
  speaker = "[S1]"
  content = "今天我们来聊聊人工智能的发展..."

[[audio_files]]
  audio_file = "/path/to/segment_02_SS2.mp3"
  image_file = "/path/to/image2.jpg"  # 用户需要填写
  speaker = "[S2]"
  content = "很高兴能参与这个讨论..."
```

## 🏗️ 项目结构

```
aiblog/
├── cmd/                    # CLI命令
│   ├── root.go            # 根命令
│   ├── generate.go        # 生成命令
│   ├── batch.go          # 批量生成命令
│   └── compose.go        # 视频合成命令
├── pkg/                   # 核心包
│   ├── blog/             # 博客生成器
│   ├── client/           # API客户端
│   ├── composer/         # 视频合成器
│   ├── config/           # 配置管理
│   ├── ffmpeg/           # FFmpeg处理器
│   ├── imagegen/         # 图片生成
│   ├── images/           # 图片管理
│   ├── mapping/          # TOML映射
│   └── tts/              # 文本转语音
├── config.yaml           # 主配置文件
├── config.example.yaml   # 配置示例
└── README.md            # 项目文档
```

## 📋 使用场景

### 🎓 教育内容创作
- 制作AI教学播客
- 生成知识讲解视频
- 创建互动学习材料

### 📰 媒体内容生产
- 快速生成新闻播客
- 制作访谈节目
- 创建专题报道

### 🎪 娱乐内容
- 生成故事播客
- 创建角色对话
- 制作有声书

### 💼 企业应用
- 产品介绍视频
- 培训材料制作
- 会议记录转播客

## 🔧 高级功能

### 自定义对话模板

修改 `pkg/client/prompts.go` 中的提示词模板，定制对话风格：

```go
const DialoguePrompt = `
你是一位专业的播客主持人，请生成深度对话...
要求：
1. 内容专业且易懂
2. 体现嘉宾观点差异
3. 包含实际案例
...
`
```

### 批量主题定制

修改 `pkg/blog/generator.go` 中的主题列表：

```go
topics := []string{
    "您的自定义主题1",
    "您的自定义主题2",
    ...
}
```

### 扩展语音模型

在 `pkg/tts/tts.go` 中添加新的TTS服务提供商支持。

## 🤝 贡献指南

我们欢迎所有形式的贡献！

### 开发环境设置

1. Fork项目
2. 创建功能分支
```bash
git checkout -b feature/your-feature
```
3. 提交更改
```bash
git commit -am "Add some feature"
```
4. 推送分支
```bash
git push origin feature/your-feature
```
5. 创建Pull Request

### 代码规范
- 遵循Go官方代码规范
- 添加必要的注释
- 编写单元测试
- 更新相关文档

## 📄 许可证

本项目采用 [MIT许可证](LICENSE)。

## 🙏 致谢

- [SiliconFlow](https://siliconflow.cn) - 提供强大的AI API服务
- [FFmpeg](https://ffmpeg.org) - 音视频处理核心
- [Cobra](https://github.com/spf13/cobra) - CLI框架
- [Viper](https://github.com/spf13/viper) - 配置管理

## 📞 联系我们

- 项目主页: https://github.com/yourusername/aiblog
- 问题反馈: https://github.com/yourusername/aiblog/issues
- 邮箱: your.email@example.com

---

⭐ 如果这个项目对您有帮助，请给我们一个星标！