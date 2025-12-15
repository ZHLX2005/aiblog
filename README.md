# AI博客生成器

这是一个智能的AI博客生成工具，可以根据指定的主题自动生成包含对话内容的博客，并将对话转换为不同语音的音频文件，最终合成完整的播客音频。

## 功能特点

- 🤖 **智能对话生成**：使用SiliconFlow的DeepSeek-V3模型生成有趣的双人对话
- 🎤 **多种语音支持**：支持两种不同的语音角色（主持人和嘉宾）
- 🎵 **音频合成**：使用FFmpeg将多个音频片段合并成完整的播客
- 📝 **批量生成**：支持一次性生成多个不同主题的博客
- ⚙️ **灵活配置**：通过YAML配置文件和环境变量进行配置

## 项目结构

```
9000-aiblog/
├── cmd/                 # 命令行接口
│   ├── root.go         # 根命令
│   ├── generate.go     # 生成单个博客命令
│   └── batch.go        # 批量生成命令
├── pkg/                # 核心功能包
│   ├── config/         # 配置管理
│   ├── client/         # API客户端
│   ├── tts/            # 文本转语音
│   ├── ffmpeg/         # 音频处理
│   └── blog/           # 博客生成器
├── config.yaml         # 配置文件
├── go.mod              # Go模块文件
└── main.go             # 主程序入口
```

## 快速开始

### 1. 配置

编辑 `config.yaml` 文件：

```yaml
siliconflow:
  api_key: "your-api-key-here"
  base_url: "https://api.siliconflow.cn/v1"
  chat_model: "deepseek-ai/DeepSeek-V3.2-Exp"
  tts_model: "fnlp/MOSS-TTSD-v0.5"

tts:
  voice1: "fnlp/MOSS-TTSD-v0.5:alex"  # 主持人语音
  voice2: "fnlp/MOSS-TTSD-v0.5:anna"  # 嘉宾语音
  sample_rate: 32000
  speed: 1.0
  format: "mp3"

ffmpeg:
  path: "ffmpeg"  # 确保ffmpeg在PATH中

output:
  dir: "./output"
  audio_dir: "./output/audio"
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 编译

```bash
go build -o aiblog.exe .
```

### 4. 运行

#### 生成单个博客

```bash
./aiblog.exe generate -t "人工智能的未来发展趋势"
```

#### 批量生成博客

```bash
./aiblog.exe batch -c 3
```

## 使用说明

### 命令行参数

#### `generate` 命令
生成单个AI博客：

```bash
aiblog generate -t <主题>
```

参数：
- `-t, --topic`: 博客主题（必需）

#### `batch` 命令
批量生成多个博客：

```bash
aiblog batch -c <数量>
```

参数：
- `-c, --count`: 生成的博客数量（1-10，默认5）

### 预定义主题

程序包含以下预定义主题用于批量生成：
1. 人工智能的未来发展趋势
2. 区块链技术在金融领域的应用
3. 远程工作的挑战与机遇
4. 可持续发展与环保创新
5. 数字时代的隐私保护
6. 虚拟现实与元宇宙
7. 量子计算的商业前景
8. 生物技术的伦理考量
9. 太空探索的新篇章
10. 教育科技的革命性变革

### 输出文件

生成的内容保存在 `./output` 目录中：
- `<timestamp>.txt`: 对话文本
- `podcast_<主题>_<timestamp>.mp3`: 合成的音频文件
- `batch_report.txt`: 批量生成的报告

## 工作流程

1. **文本生成**：使用SiliconFlow API生成主持人和嘉宾的对话
2. **解析对话**：提取[S1]和[S2]标记的对话段落
3. **语音合成**：为每个段落生成对应的语音音频
4. **音频合并**：使用FFmpeg将所有音频片段合并成完整播客
5. **文件保存**：保存文本和音频文件

## 依赖要求

- Go 1.21+
- FFmpeg（需要在PATH中）
- SiliconFlow API密钥

## 环境变量

可以通过环境变量覆盖配置：

```bash
export AIBLOG_SILICONFLOW_API_KEY="your-api-key"
export AIBLOG_OUTPUT_DIR="./my-output"
```

## 注意事项

- 确保FFmpeg已安装并可在PATH中访问
- API密钥需要有效额度
- 批量生成时注意API调用限制
- 程序会自动创建输出目录

## 故障排除

1. **FFmpeg未找到**：确保ffmpeg已安装并在PATH中
2. **API调用失败**：检查API密钥是否正确
3. **权限错误**：确保有写入输出目录的权限